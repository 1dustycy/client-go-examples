package cronjob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/appscode/go/wait"
	"github.com/betterchen/client-go-examples/pkg/util"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	discoveryutil "kmodules.xyz/client-go/discovery"
)

func createOrUpdateFromReader(
	discoveryClient discovery.DiscoveryInterface,
	dynamicClient dynamic.Interface,
	transform func(oldObj *unstructured.Unstructured, newObj *unstructured.Unstructured) *unstructured.Unstructured,
	reader io.Reader,
) error {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	for {
		// unmarshals the next object from the underlying stream into the provide object
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("failed to decode the next object from the underlying stream into an unstructured object: %v", err)
			return err
		}

		// find the object's resource interface
		gvk := obj.GroupVersionKind()
		gvr, err := discoveryutil.ResourceForGVK(discoveryClient, gvk)
		if err != nil {
			log.Printf("failed to discovery GVR for the resource %v: %v", gvk, err)
			return err
		}
		namespace := obj.GetNamespace()
		ri := dynamicClient.Resource(gvr).Namespace(namespace)

		name := obj.GetName()
		kind := obj.GetKind()

		// handle the object using its resource interface
		err = wait.PollImmediate(50*time.Millisecond, 2*time.Second, func() (bool, error) {
			oldObj, err := ri.Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				if !kapierrors.IsNotFound(err) {
					msg := fmt.Sprintf("failed to retrieve current configuration of the %s %s/%s: %v", kind, namespace, name, err)
					return false, errors.New(msg)
				}

				// create it because the resource is not existed
				obj.SetResourceVersion("")
				if transform != nil {
					obj = transform(nil, obj.DeepCopy())
				}
				created, err := ri.Create(context.Background(), obj, metav1.CreateOptions{})
				if err != nil {
					log.Printf("failed to create the %s resource %s/%s: %v", kind, namespace, name, err)
					return false, err
				}
				log.Printf("created the resource: %v", created)
				return true, nil
			}
			// found the old resource, so we update it
			obj.SetResourceVersion(oldObj.GetResourceVersion())
			if transform != nil {
				obj = transform(oldObj.DeepCopy(), obj.DeepCopy())
			}

			// found the old resource, so we update it
			patchData, err := obj.MarshalJSON()
			if err != nil {
				log.Printf("failed to marshal obj to []byte, err is %v", err)
				return false, err
			}

			updated, err := ri.Patch(context.Background(), obj.GetName(), types.MergePatchType, patchData, metav1.PatchOptions{})
			if err != nil {
				if kapierrors.IsForbidden(err) {
					log.Printf("failed to update resource: %v", err)
					return false, nil
				}
				log.Printf("failed to update the existed %s resource %s/%s, %v", kind, namespace, name, err)
				return false, err
			}
			log.Printf("updated the resource: %v", updated)
			return true, nil
		})
		if err != nil {
			log.Printf("failed to deploy the %s resource %s/%s: %v", kind, namespace, name, err)
			return err
		}
	}
	log.Printf("deployed all resources")
	return nil
}

// CreateOrUpdateCronJobByYAML .
func CreateOrUpdateCronJobByYAML(cli util.ClientInterface, filepath string) error {
	if cli == nil {
		return errors.New("client is nil")
	}

	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	tee := io.TeeReader(f, &buf)

	decoder := yaml.NewYAMLOrJSONDecoder(tee, 4096)
	obj := &unstructured.Unstructured{}
	if err := decoder.Decode(obj); err != nil {
		return fmt.Errorf("failed to decode yaml: %v", err)
	}

	log.Print(obj)

	if obj.GetKind() != "CronJob" {
		return fmt.Errorf("kind must be cronjob, got: %s", obj.GetKind())
	}

	if err := createOrUpdateFromReader(
		cli.Discovery(),
		cli,
		nil,
		&buf,
	); err != nil {
		return fmt.Errorf("failed to create or update cronjob: %v", err)
	}

	return nil
}

// GetCronJob .
func GetCronJob(cli util.ClientInterface, namespace, name string) (cj *v1beta1.CronJob, err error) {
	return cli.BatchV1beta1().CronJobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// GetCronJobEvents .
func GetCronJobEvents(cli util.ClientInterface, namespace, name string) (events *corev1.EventList, err error) {

	cj, err := cli.BatchV1beta1().CronJobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return
	}

	return cli.CoreV1().Events(namespace).Search(scheme.Scheme, cj)
}

// ListCronJob .
func ListCronJob(cli util.ClientInterface, namespace string) (cjs *v1beta1.CronJobList, err error) {
	return cli.BatchV1beta1().CronJobs(namespace).List(context.Background(), metav1.ListOptions{})
}

// DeleteCronJob .
func DeleteCronJob(cli util.ClientInterface, namespace, name string) error {
	return cli.BatchV1beta1().CronJobs(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}
