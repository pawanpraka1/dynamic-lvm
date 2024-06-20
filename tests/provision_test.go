/*
Copyright 2021 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = Describe("[lvmpv] TEST VOLUME PROVISIONING", func() {
	Context("App is deployed with lvm driver", func() {
		It("Running volume Creation Tests", volumeCreationTest)
		It("Running volume/snapshot Capacity Tests", capacityTest)
	})
})

func deleteAppAndPvc(appnames []string, pvcname string) {
	for _, appName := range appnames {
		By("Deleting the application deployment " + appName)
		deleteAppDeployment(appName)
	}
	By("Deleting the PVC")
	deleteAndVerifyPVC(pvcName)
}

func fsVolCreationTest() {
	fstypes := []string{"ext4", "xfs", "btrfs"}
	for _, fstype := range fstypes {
		By("####### Creating the storage class : " + fstype + " #######")
		createFstypeStorageClass(fstype)
		By("creating and verifying PVC bound status", createAndVerifyPVC)
		By("Creating and deploying app pod", createDeployVerifyApp)
		By("verifying LVMVolume object", VerifyLVMVolume)

		resizeAndVerifyPVC(true, "8Gi")
		// do not resize after creating the snapshot(not supported)
		createSnapshot(pvcName, snapName, snapYAML)
		verifySnapshotCreated(snapName)

		if fstype != "btrfs" {
			// if snapshot is there, resize should fail
			resizeAndVerifyPVC(false, "10Gi")
		}

		deleteAppAndPvc(appNames, pvcName)

		// PV should be present after PVC deletion since snapshot is present
		By("Verifying that PV exists after PVC deletion")
		verifyPVForPVC(true, pvcName)

		deleteSnapshot(pvcName, snapName, snapYAML)

		By("Verifying that PV is deleted after snapshot deletion")
		verifyPVForPVC(false, pvcName)
		By("Deleting storage class", deleteStorageClass)
	}
}

func blockVolCreationTest() {
	By("Creating default storage class", createStorageClass)
	By("creating and verifying PVC bound status", createAndVerifyBlockPVC)
	By("Creating and deploying app pod", createDeployVerifyBlockApp)
	By("verifying LVMVolume object", VerifyLVMVolume)
	By("Online resizing the block volume")
	resizeAndVerifyPVC(true, "8Gi")
	By("create snapshot")
	createSnapshot(pvcName, snapName, snapYAML)
	By("verify snapshot")
	verifySnapshotCreated(snapName)
	deleteAppAndPvc(appNames, pvcName)
	By("Verifying that PV exists after PVC deletion")
	verifyPVForPVC(true, pvcName)
	By("Deleting snapshot")
	deleteSnapshot(pvcName, snapName, snapYAML)
	By("Verifying that PV is deleted after snapshot deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func sharedVolumeTest() {
	By("Creating shared LV storage class", createSharedVolStorageClass)
	By("creating and verifying PVC bound status", createAndVerifyPVC)
	//we use two fio app pods for this test.
	appNames = append(appNames, "fio-ci-1")
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object", VerifyLVMVolume)
	By("Online resizing the shared volume")
	resizeAndVerifyPVC(true, "8Gi")
	deleteAppAndPvc(appNames, pvcName)
	By("Deleting storage class", deleteStorageClass)
	// Reset the app list back to original
	appNames = appNames[:len(appNames)-1]
}

func thinVolCreationTest() {
	By("Creating thinProvision storage class", createThinStorageClass)
	By("creating and verifying PVC bound status", createAndVerifyPVC)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object", VerifyLVMVolume)
	By("Online resizing the block volume")
	resizeAndVerifyPVC(true, "8Gi")
	By("create snapshot")
	createSnapshot(pvcName, snapName, snapYAML)
	By("verify snapshot")
	verifySnapshotCreated(snapName)
	deleteAppAndPvc(appNames, pvcName)
	By("Verifying that PV exists after PVC deletion")
	verifyPVForPVC(true, pvcName)
	By("Deleting snapshot")
	deleteSnapshot(pvcName, snapName, snapYAML)
	By("Verifying that PV is deleted after snapshot deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting thinProvision storage class", deleteStorageClass)
}

func thinVolCapacityTest() {
	By("Creating thinProvision storage class", createThinStorageClass)
	By("creating and verifying PVC bound status", createAndVerifyPVC)
	By("enabling monitoring on thinpool", enableThinpoolMonitoring)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying thinpool auto-extended", VerifyThinpoolExtend)
	By("verifying LVMVolume object", VerifyLVMVolume)
	deleteAppAndPvc(appNames, pvcName)
	By("Deleting thinProvision storage class", deleteStorageClass)
}

func sizedSnapFSTest() {
	createFstypeStorageClass("ext4")
	By("creating and verifying PVC bound status", createAndVerifyPVC)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object", VerifyLVMVolume)
	createSnapshot(pvcName, snapName, sizedsnapYAML)
	verifySnapshotCreated(snapName)
	deleteAppAndPvc(appNames, pvcName)
	deleteSnapshot(pvcName, snapName, sizedsnapYAML)
	By("Deleting storage class", deleteStorageClass)
}

func sizedSnapBlockTest() {
	By("Creating default storage class", createStorageClass)
	By("creating and verifying PVC bound status", createAndVerifyPVC)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object", VerifyLVMVolume)
	createSnapshot(pvcName, snapName, sizedsnapYAML)
	verifySnapshotCreated(snapName)
	deleteAppAndPvc(appNames, pvcName)
	deleteSnapshot(pvcName, snapName, sizedsnapYAML)
	By("Deleting storage class", deleteStorageClass)
}

func sizedSnapshotTest() {
	By("Sized snapshot for filesystem volume", sizedSnapFSTest)
	By("Sized snapshot for block volume", sizedSnapBlockTest)
}

func leakProtectionTest() {
	By("Creating default storage class", createStorageClass)
	ds := deleteNodeDaemonSet() // ensure that provisioning remains in pending state.

	By("Creating PVC", createPVC)
	time.Sleep(30 * time.Second) // wait for external provisioner to pick up new pvc
	By("Verify pending lvm volume resource")
	verifyPendingLVMVolume(getGeneratedVolName(pvcObj))

	existingSize := scaleControllerPlugin(0) // remove the external provisioner
	createNodeDaemonSet(ds)                  // provision the volume now by restoring node plugin
	By("Wait for lvm volume resource to become ready", WaitForLVMVolumeReady)

	deleteAndVerifyLeakedPVC(pvcName)
	scaleControllerPlugin(existingSize)

	gomega.Expect(IsPVCDeletedEventually(pvcName)).To(gomega.Equal(true),
		"failed to garbage collect leaked pvc")
	By("Deleting storage class", deleteStorageClass)
}

// This acts as a test just to call the new wrapper functions,
// Once we write tests calling them this will not be required.
// This doesnt test any Openebs component.
func lvmOps() {
	device := createPV(6)
	createVg("newvgcode3", device)
	device_1 := createPV(5)
	extendVg("newvgcode3", device_1)
	removeVg("newvgcode3")
	removePV(device)
	removePV(device_1)
}

func volumeCreationTest() {
	By("Running filesystem volume creation test", fsVolCreationTest)
	By("Running block volume creation test", blockVolCreationTest)
	By("Running thin volume creation test", thinVolCreationTest)
	By("Running leak protection test", leakProtectionTest)
	By("Running shared volume for two app pods on same node test", sharedVolumeTest)
	By("Running Lvm Ops", lvmOps)
}

func capacityTest() {
	By("Running thin volume capacity test", thinVolCapacityTest)
	By("Running sized snapshot test", sizedSnapshotTest)
}
