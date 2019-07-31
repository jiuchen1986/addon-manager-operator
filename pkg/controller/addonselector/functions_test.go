package addonselector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	addonmanagerv1alpha1 "github.com/jiuchen1986/addon-manager-operator/pkg/apis/addonmanager/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	//"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var defaultInstanceId string = "addon-manager-operator"

func TestSetStatus(t *testing.T) {

	object1 := addonmanagerv1alpha1.AddonObject{
		Namespace: "test-ns",
		Name:      "test-pod-1",
		Group:     "",
		Kind:      "Pod",
	}

	object2 := addonmanagerv1alpha1.AddonObject{
		Namespace: "test-ns",
		Name:      "test-pod-2",
		Group:     "",
		Kind:      "Pod",
	}

	selector := &addonmanagerv1alpha1.AddonSelector{
		Spec: addonmanagerv1alpha1.AddonSelectorSpec{
			Addons: []addonmanagerv1alpha1.Addon{{
				Name:         "test-addon-1",
				AddonObjects: []addonmanagerv1alpha1.AddonObject{object1, object2},
			}, {
				Name:         "test-addon-2",
				AddonObjects: []addonmanagerv1alpha1.AddonObject{object1, object2},
			}},
		},
	}

	err := setAddonObjectStatus(selector, "test-addon-1", defaultInstanceId, object1, false)

	if err != nil {
		t.Error(err)
	}

	instanceAddonStatus, ok := selector.Status.InstanceAwareAddonStatuses[defaultInstanceId]

	if !ok {
		t.Error("Add object1 status to selector failed!")
	}

	if instanceAddonStatus[0].AddonName != "test-addon-1" {
		t.Error("The first addon status is wrongly added. The addon's name should be \"test-addon-1\"!")
	}

	if !reflect.DeepEqual(instanceAddonStatus[0].AddonObjectStatuses[0].AddonObject, object1) {
		t.Error("Wrongly add firt object1's status. The object should equal to object1!")
	}

	if instanceAddonStatus[0].AddonObjectStatuses[0].Protect {
		t.Error("The first added object's status is wrong. The \"Protect\" field should be false!")
	}

	err = setAddonObjectStatus(selector, "test-addon-1", defaultInstanceId, object1, true)

	if err != nil {
		t.Error(err)
	}

	if len(instanceAddonStatus[0].AddonObjectStatuses) != 1 {
		t.Error("Update the object1's status should not change the total number of all object status!")
	}

	if !reflect.DeepEqual(instanceAddonStatus[0].AddonObjectStatuses[0].AddonObject, object1) {
		t.Error("Wrongly update object1's status. The object should equal to object1!")
	}

	if !instanceAddonStatus[0].AddonObjectStatuses[0].Protect {
		t.Error("Wrongly update object1's status. The \"Protect\" field should be true!")
	}

	err = setAddonObjectStatus(selector, "test-addon-1", defaultInstanceId, object2, true)

	if err != nil {
		t.Error(err)
	}

	if len(instanceAddonStatus[0].AddonObjectStatuses) != 2 {
		t.Error("After adding the object2's status, the total number of all object status should be 2!")
	}

	if !reflect.DeepEqual(instanceAddonStatus[0].AddonObjectStatuses[1].AddonObject, object2) {
		t.Error("Wrongly add object2's status. The object should equal to object2!")
	}

	if !instanceAddonStatus[0].AddonObjectStatuses[0].Protect {
		t.Error("Wrongly add object2's status. The \"Protect\" field should be true!")
	}

	err = setAddonObjectStatus(selector, "test-addon-2", defaultInstanceId, object2, true)

	if err != nil {
		t.Error(err)
	}

	instanceAddonStatus, ok = selector.Status.InstanceAwareAddonStatuses[defaultInstanceId]

	if len(instanceAddonStatus) != 2 {
		t.Error("After adding another addon's status, the total number of all addon status should be 2!")
	}

	if instanceAddonStatus[1].AddonName != "test-addon-2" {
		t.Error("The second addon status is wrongly added. The addon's name should be \"test-addon-2\"!")
	}

	if !reflect.DeepEqual(instanceAddonStatus[1].AddonObjectStatuses[0].AddonObject, object2) {
		t.Error("Wrongly add object2's status to second addon's status. The object should equal to object2!")
	}

	if !instanceAddonStatus[1].AddonObjectStatuses[0].Protect {
		t.Error("Wrongly add object2's status to second addon's status. The \"Protect\" field should be true!")
	}
}

func genExampleStructuredObject() (*appsv1.Deployment, addonmanagerv1alpha1.AddonObject, string) {
	replicas := new(int32)
	deadLine := new(int32)
	historyLimit := new(int32)
	gracePeriod := new(int64)
	*replicas = int32(1)
	*deadLine = int32(600)
	*historyLimit = int32(2)
	*gracePeriod = int64(60)
	loc, _ := time.LoadLocation("CET")
	timeValue := metav1.Date(2019, time.June, 22, 7, 51, 12, 0, loc)

	liveObject := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-addon",
			Namespace:         "default",
			SelfLink:          "/apis/extensions/v1beta1/namespaces/default/deployments/test-addon",
			UID:               types.UID("fa6201c8-8830-11e9-9264-005056010877"),
			ResourceVersion:   "3649322",
			Generation:        int64(1),
			CreationTimestamp: metav1.Now(),
			Labels:            map[string]string{"app": "test-addon", "release": "test-addon"},
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision":                "1",
				"kubectl.kubernetes.io/last-applied-configuration": "hahahahaha",
			},
		},

		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-addon"},
			},
			ProgressDeadlineSeconds: deadLine,
			RevisionHistoryLimit:    historyLimit,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{},
					Labels:            map[string]string{"app": "test-addon"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "nginx",
						Image:           "nginx:1.7.9",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(80),
							Protocol:      corev1.ProtocolTCP,
						}},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						Resources:                corev1.ResourceRequirements{},
					}},
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 "default-scheduler",
					SecurityContext:               &corev1.PodSecurityContext{},
					TerminationGracePeriodSeconds: gracePeriod,
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas:  int32(1),
			ObservedGeneration: int64(1),
			Replicas:           int32(1),
			ReadyReplicas:      int32(1),
			UpdatedReplicas:    int32(1),
			Conditions: []appsv1.DeploymentCondition{
				{LastTransitionTime: timeValue,
					LastUpdateTime: timeValue,
					Message:        "Deployment has minimum availability.",
					Reason:         "MinimumReplicasAvailable",
					Status:         corev1.ConditionTrue,
					Type:           appsv1.DeploymentAvailable},
				{LastTransitionTime: timeValue,
					LastUpdateTime: timeValue,
					Message:        "ReplicaSet \"test-addon-cbf9fc466\" has successfully progressed.",
					Reason:         "NewReplicaSetAvailable",
					Status:         corev1.ConditionTrue,
					Type:           appsv1.DeploymentProgressing},
			},
		},
	}

	addonName := "test-addon"

	addonObj := addonmanagerv1alpha1.AddonObject{
		Namespace: "default",
		Name:      "test-addon",
		Group:     "apps",
		Kind:      "Deployment",
	}
	return liveObject, addonObj, addonName
}

type testUnstructured struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Size              int32        `json:"size,omitempty"`
	Name              string       `json:"name,omitempty"`
	Data              dataStruct   `json:"data,omitempty"`
	Status            statusStruct `json:"status,omitempty"`
}

type dataStruct struct {
	Data []byte `json:"data,omitempty"`
}

type statusStruct struct {
	Status []string `json:"status,omitempty"`
}

func genExampleUnstructuredObject() (*unstructured.Unstructured, addonmanagerv1alpha1.AddonObject, string) {
	addonObj := addonmanagerv1alpha1.AddonObject{
		Name:      "test-unstructured",
		Namespace: "test",
		Group:     "www.example.com",
		Version:   "v1",
		Kind:      "ExampleKind",
	}

	ut := testUnstructured{
		TypeMeta: metav1.TypeMeta{
			Kind:       addonObj.Kind,
			APIVersion: fmt.Sprintf("%s/%s", addonObj.Group, addonObj.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      addonObj.Name,
			Namespace: addonObj.Namespace,
		},
		Size: int32(4),
		Name: "HaHa",
		Data: dataStruct{
			Data: []byte("Test data here!"),
		},
		Status: statusStruct{
			Status: []string{"Xiaoming", "Xiaohong", "Xiaolan"},
		},
	}

	jsonByte, _ := json.Marshal(ut)
	var out map[string]interface{}
	json.Unmarshal(jsonByte, &out)

	liveObject := &unstructured.Unstructured{Object: out}

	addonName := "test-addon"

	return liveObject, addonObj, addonName
}

func TestGenUnstructuredObject(t *testing.T) {
	addonObj := addonmanagerv1alpha1.AddonObject{
		Name:      "test-unstructured",
		Namespace: "default",
		Group:     "www.example.com",
		Version:   "v1",
		Kind:      "ExampleKind",
	}

	unstructuredObj, err := genRuntimeObject(addonObj, scheme.Scheme)
	if err != nil {
		t.Errorf("Failed to generate unstructured object: %s!", err.Error())
	}

	if _, ok := unstructuredObj.(*unstructured.Unstructured); !ok {
		t.Error("Failed to generate unstructured object, generated object should be a pointer to unstructured.Unstructured!")
	}

	if _, ok := unstructuredObj.(metav1.Object); !ok {
		t.Error("Unstructured object doesn't implement metav1.Object!")
	}
}

func TestGenListObject(t *testing.T) {
	addonObj := addonmanagerv1alpha1.AddonObject{
		Namespace: "default",
		Name:      "test-secret",
		Group:     "",
		Kind:      "Secret",
	}
	listObj, obj, err := genListObject(addonObj, scheme.Scheme)
	if err != nil {
		t.Errorf("Failed to generate list object from a Secret GVK: %s", err)
	}
	if reflect.TypeOf(obj) != reflect.TypeOf(&corev1.Secret{}) {
		t.Errorf("Wrongly generate object from a Secret GVK, get: %s, want: %s", reflect.TypeOf(obj), reflect.TypeOf(&corev1.Secret{}))
	}
	// fmt.Println(reflect.TypeOf(listObj))
	if reflect.TypeOf(listObj) != reflect.TypeOf(&corev1.SecretList{}) {
		t.Errorf("Wrongly generate list object from a Secret GVK, get: %s, want: %s", reflect.TypeOf(listObj), reflect.TypeOf(&corev1.SecretList{}))
	}

	addonObj = addonmanagerv1alpha1.AddonObject{
		Namespace: "default",
		Name:      "test-deploy",
		Group:     "apps",
		Kind:      "Deployment",
		Version:   "v1",
	}
	listObj, obj, err = genListObject(addonObj, scheme.Scheme)
	if err != nil {
		t.Errorf("Failed to generate list object from a Deployment GVK: %s", err)
	}
	if reflect.TypeOf(obj) != reflect.TypeOf(&appsv1.Deployment{}) {
		t.Errorf("Wrongly generate object from a Deployment GVK, get: %s, want: %s", reflect.TypeOf(obj), reflect.TypeOf(&appsv1.Deployment{}))
	}
	// fmt.Println(reflect.TypeOf(listObj))
	if reflect.TypeOf(listObj) != reflect.TypeOf(&appsv1.DeploymentList{}) {
		t.Errorf("Wrongly generate list object from a Deployment GVK, get: %s, want: %s", reflect.TypeOf(listObj), reflect.TypeOf(&appsv1.DeploymentList{}))
	}
}

func TestGetInstanceFromListObjByNamePrefix(t *testing.T) {
        addonObj := addonmanagerv1alpha1.AddonObject{
                Namespace: "default",
                Name:      "test-prefix-",
                Group:     "",
                Kind:      "Secret",
                Version:   "v1",
                IsNamePrefix: true,
        }
        fullName := "test-prefix-kdxldk"
        listObj := &corev1.SecretList{
                Items: []corev1.Secret{
                        corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "lalala"}},
                        corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "hahaha"}},
                        corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: fullName}},
                        corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "wawawa"}},
                        corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-predss"}},
                },
        }
        checkObj := &corev1.Secret{}
        obj, err := getInstanceFromListObjByNamePrefix(listObj, checkObj, addonObj)
        if err != nil {
                t.Errorf("Failed to get instance from list object by name prefix: %s", err)
        }
        if reflect.TypeOf(obj) != reflect.TypeOf(checkObj) {
                t.Errorf("Wrongly get intance from a SecretList object, get: %s, want: %s", reflect.TypeOf(obj), reflect.TypeOf(checkObj))
        }
        metaObj, _ := obj.(metav1.Object)
        if metaObj.GetName() != fullName {
                t.Errorf("Wrongly get intance from a SecretList object, get name: %s, want name: %s", metaObj.GetName(), fullName)
        }
}

// This test relies on the test data file "testdata/serialized_test_obj.yaml"
// which contains the serialized object for protection according to the live object generated by genExampleStructuredObject
// Remember change the test data file if any logic change impacting the serialized output is applied
func TestGenObjectToProtect(t *testing.T) {
	structuredObj, addonObj, _ := genExampleStructuredObject()
	genObj, serialized, err := genObjectToProtect(structuredObj, addonObj)
	if err != nil {
		t.Errorf("Failed to generate object for protection: %s!", err.Error())
	}

	// fmt.Println(string(serialized))

	deployObj, ok := genObj.(*appsv1.Deployment)

	if !ok {
		t.Error("Failed to generate object for protection. Generated object should be a type of \"*appsv1.Deployment\"!")
	}

	if string(deployObj.GenerateName) != "" || string(deployObj.UID) != "" || string(deployObj.ResourceVersion) != "" || string(deployObj.SelfLink) != "" {
		t.Error("Failed to generate object for protection. Runtime information should be removed!")
	}

	expectOutput, er := getExpectedSerializedObj()
	if er != nil {
		t.Errorf("Failed to get expected serialized object: %s!", er.Error())
	}

	if string(serialized) != string(expectOutput) {
		// fmt.Println(string(expectOutput))
		// fmt.Println(string(serialized))
		t.Error("Wrongly serialized the output object. Output should match the test data file!")
	}
}

// This test relies on the test data file "testdata/serialized_test_unstructured_obj.yaml"
// which contains the serialized object for protection according to the live object generated by genExampleUnstructuredObject
// Remember change the test data file if any logic change impacting the serialized output is applied
func TestGenUnstructuredObjectToProtect(t *testing.T) {
	unstructuredObj, addonObj, _ := genExampleUnstructuredObject()
	genObj, serialized, err := genObjectToProtect(unstructuredObj, addonObj)
	if err != nil {
		t.Errorf("Failed to generate object for protection: %s!", err.Error())
	}

	_, ok := genObj.(*unstructured.Unstructured)

	if !ok {
		t.Error("Failed to generate unstructured object for protection. Generated object should be a type of \"*unstructured.Unstructured\"!")
	}

	expectOutput, er := getExpectedSerializedUnstructuredObj()
	if er != nil {
		t.Errorf("Failed to get expected serialized unstructured object: %s!", er.Error())
	}

	if string(serialized) != string(expectOutput) {
		fmt.Println(string(serialized))
		t.Error("Wrongly serialized the output unstructured object. Output should match the test data file!")
	}

}

func TestWriteObjectToDisk(t *testing.T) {
	structuredObj, addonObj, addonName := genExampleStructuredObject()
	_, serialized, _ := genObjectToProtect(structuredObj, addonObj)

	workDir, er := getAbsWorkDir()
	if er != nil {
		t.Errorf("Failed to get working directory: %s!", er.Error())
	}

	if _, er := writeObjectToDisk(serialized, addonName, workDir, addonObj); er != nil {
		t.Errorf("Failed to write object to disk: %s!", er.Error())
	}

	outputFile := filepath.Join(workDir, genManifestFileName(addonName, addonObj))
	outputSerialized, err := ioutil.ReadFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read the output file: %s!", err.Error())
	}

	if string(outputSerialized) != string(serialized) {
		t.Error("Wrongly write serialized object to disk!")
	}

	if err := os.Remove(outputFile); err != nil {
		t.Errorf("Failed to remove the output file: %s!", err.Error())
	}
}

func getAbsWorkDir() (string, error) {
	workDir, er := os.Getwd()
	if er != nil {
		return "", er
	}
	workDir, er = filepath.Abs(workDir)
	if er != nil {
		return "", er
	}

	return workDir, nil
}

func getExpectedSerializedObj() ([]byte, error) {
	workDir, er := getAbsWorkDir()
	if er != nil {
		return nil, er
	}

	filename := filepath.Join(workDir, "testdata", "serialized_test_obj.yaml")
	serialized, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return serialized, nil
}

func getExpectedSerializedUnstructuredObj() ([]byte, error) {
	workDir, er := getAbsWorkDir()
	if er != nil {
		return nil, er
	}

	filename := filepath.Join(workDir, "testdata", "serialized_test_unstructured_obj.yaml")
	serialized, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return serialized, nil
}
