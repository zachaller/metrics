package memory

import (
	"context"
	"errors"
	"fmt"
	"github.com/argoproj/metrics/pkg/kubeclient"
	"github.com/argoproj/metrics/pkg/metricproviders/prometheus"
	"k8s.io/apimachinery/pkg/util/json"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	prometheusv1 "github.com/argoproj/metrics/pkg/apis/prometheus/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

// ErrObjectNotExists means the file doesn't actually exist.
var ErrObjectNotExists = fmt.Errorf("object doesn't exist")

// ErrNamespaceNotExists means the directory for the namespace doesn't actually exist.
var ErrNamespaceNotExists = errors.New("namespace does not exist")

var _ rest.StandardStorage = &memoryREST{}
var _ rest.Scoper = &memoryREST{}
var _ rest.Storage = &memoryREST{}

// NewMemoryREST instantiates a new REST storage.
func NewMemoryREST(
	groupResource schema.GroupResource,
	codec runtime.Codec,
	isNamespaced bool,
	newFunc func() runtime.Object,
	newListFunc func() runtime.Object,
) rest.Storage {
	objRoot := filepath.Join(groupResource.Group, groupResource.Resource)

	rest := &memoryREST{
		TableConvertor: rest.NewDefaultTableConvertor(groupResource),
		codec:          codec,
		objRootPath:    objRoot,
		isNamespaced:   isNamespaced,
		newFunc:        newFunc,
		newListFunc:    newListFunc,
		watchers:       make(map[int]*jsonWatch, 10),
		storage:        make(map[string]memoryStorage, 10),
	}
	return rest
}

type memoryREST struct {
	rest.TableConvertor
	codec        runtime.Codec
	objRootPath  string
	isNamespaced bool

	muWatchers sync.RWMutex
	watchers   map[int]*jsonWatch

	newFunc     func() runtime.Object
	newListFunc func() runtime.Object

	muStorage sync.RWMutex
	storage   map[string]memoryStorage
}

type memoryStorage struct {
	obj         runtime.Object
	createdTime time.Time
}

func (f *memoryREST) notifyWatchers(ev watch.Event) {
	f.muWatchers.RLock()
	for _, w := range f.watchers {
		w.ch <- ev
	}
	f.muWatchers.RUnlock()
}

func (f *memoryREST) New() runtime.Object {
	return f.newFunc()
}

func (f *memoryREST) Destroy() {}

func (f *memoryREST) NewList() runtime.Object {
	return f.newListFunc()
}

func (f *memoryREST) NamespaceScoped() bool {
	return f.isNamespaced
}

func (f *memoryREST) Get(
	ctx context.Context,
	name string,
	options *metav1.GetOptions,
) (runtime.Object, error) {

	key := f.objectMemoryKey(ctx, name)
	f.muStorage.RLock()

	obj, found := f.storage[key]
	if !found {
		return &metav1.Status{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Status:   metav1.StatusFailure,
			Message:  fmt.Sprintf("object %s not found", name),
			Reason:   "",
			Details:  nil,
			Code:     404,
		}, nil
	}
	mqr := obj.obj.(*prometheusv1.MetricQueryRun).DeepCopy()
	f.muStorage.RUnlock()

	_, dynamicClient, err := kubeclient.NewKubeClient()
	if err != nil {
		return nil, err
	}

	v := prometheusv1.MetricQuery{}
	un, err := dynamicClient.Resource(v.GetGroupVersionResource()).Namespace(mqr.Namespace).Get(ctx, mqr.ObjectMeta.OwnerReferences[0].Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	jsonBytes, err := un.MarshalJSON()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, &v)
	if err != nil {
		return nil, err
	}

	pClient, err := prometheus.NewPrometheus(v.Spec.Address)
	if err != nil {
		return nil, err
	}

	timeLength, err := time.ParseDuration(v.Spec.TimeLength)
	if err != nil {
		return nil, err
	}

	step, err := time.ParseDuration(v.Spec.Step)
	if err != nil {
		return nil, err
	}

	res, err := pClient.Query(ctx, v.Spec.Query, time.Now().Add(-timeLength), time.Now(), step)
	if err != nil {
		return nil, err
	}

	mqr.Spec.Result = res

	return mqr, nil
}

func (f *memoryREST) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	newListObj := f.NewList()
	v, err := getListPrt(newListObj)
	if err != nil {
		return nil, err
	}

	namespace := genericapirequest.NamespaceValue(ctx)
	f.muStorage.RLock()
	for key, value := range f.storage {
		if f.isNamespaced {
			if !strings.HasPrefix(key, namespace) {
				continue
			}
		}
		appendItem(v, value.obj)
	}
	f.muStorage.RUnlock()

	return newListObj, nil
}

func (f *memoryREST) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions,
) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	accessor.SetCreationTimestamp(metav1.Now())
	key := f.objectMemoryKey(ctx, accessor.GetName())

	if f.isNamespaced {
		// ensures namespace dir
		_, ok := genericapirequest.NamespaceFrom(ctx)
		if !ok {
			return nil, ErrNamespaceNotExists
		}
	}

	f.muStorage.Lock()
	f.storage[key] = memoryStorage{
		obj:         obj,
		createdTime: time.Now(),
	}
	f.muStorage.Unlock()

	f.notifyWatchers(watch.Event{
		Type:   watch.Added,
		Object: obj,
	})

	return obj, nil
}

func (f *memoryREST) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions,
) (runtime.Object, bool, error) {
	isCreate := false
	oldObj, err := f.Get(ctx, name, nil)
	if err != nil {
		if !forceAllowCreate {
			return nil, false, err
		}
		isCreate = true
	}

	// TODO: should not be necessary, verify Get works before creating filepath
	if f.isNamespaced {
		// ensures namespace dir
		_, ok := genericapirequest.NamespaceFrom(ctx)
		if !ok {
			return nil, false, ErrNamespaceNotExists
		}
	}

	updatedObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}
	key := f.objectMemoryKey(ctx, name)

	if isCreate {
		if createValidation != nil {
			if err := createValidation(ctx, updatedObj); err != nil {
				return nil, false, err
			}
		}

		f.muStorage.Lock()
		f.storage[key] = memoryStorage{
			obj:         updatedObj,
			createdTime: time.Now(),
		}
		f.muStorage.Unlock()

		f.notifyWatchers(watch.Event{
			Type:   watch.Added,
			Object: updatedObj,
		})
		return updatedObj, true, nil
	}

	if updateValidation != nil {
		if err := updateValidation(ctx, updatedObj, oldObj); err != nil {
			return nil, false, err
		}
	}

	f.muStorage.Lock()
	f.storage[key] = memoryStorage{
		obj:         updatedObj,
		createdTime: time.Now(),
	}
	f.muStorage.Unlock()

	f.notifyWatchers(watch.Event{
		Type:   watch.Modified,
		Object: updatedObj,
	})
	return updatedObj, false, nil
}

func (f *memoryREST) Delete(
	ctx context.Context,
	name string,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	key := f.objectMemoryKey(ctx, name)
	_, found := f.storage[key]
	if !found {
		return &metav1.Status{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Status:   metav1.StatusFailure,
			Message:  fmt.Sprintf("object %s not found", name),
			Reason:   "",
			Details:  nil,
			Code:     404,
		}, false, nil
	}

	oldObj, err := f.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}
	if deleteValidation != nil {
		if err := deleteValidation(ctx, oldObj); err != nil {
			return nil, false, err
		}
	}

	f.muStorage.Lock()
	delete(f.storage, key)
	f.muStorage.Unlock()

	f.notifyWatchers(watch.Event{
		Type:   watch.Deleted,
		Object: oldObj,
	})
	return oldObj, true, nil
}

func (f *memoryREST) DeleteCollection(
	ctx context.Context,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions,
	listOptions *metainternalversion.ListOptions,
) (runtime.Object, error) {
	newListObj := f.NewList()
	v, err := getListPrt(newListObj)
	if err != nil {
		return nil, err
	}

	//dirname := f.objectNamespace(ctx)
	//if err := visitDir(dirname, f.newFunc, f.codec, func(path string, obj runtime.Object) {
	//	_ = os.Remove(path)
	//	appendItem(v, obj)
	//}); err != nil {
	//	return nil, fmt.Errorf("failed walking filepath %v", dirname)
	//}

	namespace := genericapirequest.NamespaceValue(ctx)
	for key, value := range f.storage {
		if f.isNamespaced {
			if !strings.HasPrefix(key, namespace) {
				continue
			}
		}
		appendItem(v, value.obj)
		delete(f.storage, key)
	}

	return newListObj, nil
}

func (f *memoryREST) objectMemoryKey(ctx context.Context, name string) string {
	if f.isNamespaced {
		// FIXME: return error if namespace is not found
		ns, _ := genericapirequest.NamespaceFrom(ctx)
		return strings.Join([]string{ns, name}, "/")
	}
	return name
}

func (f *memoryREST) objectNamespace(ctx context.Context) string {
	if f.isNamespaced {
		// FIXME: return error if namespace is not found
		ns, _ := genericapirequest.NamespaceFrom(ctx)
		return ns
	}
	return ""
}

func appendItem(v reflect.Value, obj runtime.Object) {
	v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
}

func getListPrt(listObj runtime.Object) (reflect.Value, error) {
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return reflect.Value{}, err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("need ptr to slice: %v", err)
	}
	return v, nil
}

func (f *memoryREST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	jw := &jsonWatch{
		id: len(f.watchers),
		f:  f,
		ch: make(chan watch.Event, 10),
	}
	// On initial watch, send all the existing objects
	list, err := f.List(ctx, options)
	if err != nil {
		return nil, err
	}

	danger := reflect.ValueOf(list).Elem()
	items := danger.FieldByName("Items")

	for i := 0; i < items.Len(); i++ {
		obj := items.Index(i).Addr().Interface().(runtime.Object)
		jw.ch <- watch.Event{
			Type:   watch.Added,
			Object: obj,
		}
	}

	f.muWatchers.Lock()
	f.watchers[jw.id] = jw
	f.muWatchers.Unlock()

	return jw, nil
}

type jsonWatch struct {
	f  *memoryREST
	id int
	ch chan watch.Event
}

func (w *jsonWatch) Stop() {
	w.f.muWatchers.Lock()
	delete(w.f.watchers, w.id)
	w.f.muWatchers.Unlock()
}

func (w *jsonWatch) ResultChan() <-chan watch.Event {
	return w.ch
}

// TODO: implement custom table printer optionally
// func (f *memoryREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
// 	return &metav1.Table{}, nil
// }
