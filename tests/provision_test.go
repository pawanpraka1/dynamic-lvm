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
		It("Running scheduling Tests", schedulingTest)
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

func setupVg(size int, name string) string {
	device := createPV(size)
	createVg(name, device)
	return device
}

func cleanupVg(device string, name string) {
	removeVg(name)
	removePV(device)
}

func fsVolCreationTest() {
	fstypes := []string{"ext4", "xfs", "btrfs"}
	for _, fstype := range fstypes {
		By("####### Creating the storage class : " + fstype + " #######")
		createFstypeStorageClass(fstype)
		By("Creating and verifying PVC bound status")
		createAndVerifyPVC(true)
		By("Creating and deploying app pod", createDeployVerifyApp)
		By("Verifying LVMVolume object to be Ready")
		VerifyLVMVolume(true, "")

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
	By("Creating and verifying PVC bound status")
	createAndVerifyBlockPVC(true)
	By("Creating and deploying app pod", createDeployVerifyBlockApp)
	By("Verifying LVMVolume object to be Ready")
	VerifyLVMVolume(true, "")
	By("Online resizing the block volume")
	resizeAndVerifyPVC(true, "8Gi")
	By("Creating snapshot")
	createSnapshot(pvcName, snapName, snapYAML)
	By("Verifying snapshot")
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

func vgExtendNeededForProvsioningTest() {
	device_0 := setupVg(7, "lvmvgdiff")
	device := setupVg(3, "lvmvg")
	device_1 := createPV(4)
	defer removePV(device_1)
	defer cleanupVg(device_0, "lvmvgdiff")
	defer cleanupVg(device, "lvmvg")
	By("Creating default storage class", createStorageClass)
	By("Creating and verifying PVC Not Bound status")
	createAndVerifyPVC(false)
	By("Verifying LVMVolume object to be not Ready")
	VerifyLVMVolume(false, "")
	extendVg("lvmvg", device_1)
	By("Verifying PVC bound status after vg extend")
	VerifyBlockPVC()
	By("Verifying LVMVolume object to be Ready after vg extend")
	VerifyLVMVolume(true, "")
	By("Deleting pvc")
	deleteAndVerifyPVC(pvcName)
	By("Verifying that PV doesnt exists after PVC deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func vgPatternMatchPresentTest() {
	device := setupVg(20, "lvmvg112")
	device_1 := setupVg(20, "lvmvg")
	defer cleanupVg(device_1, "lvmvg")
	defer cleanupVg(device, "lvmvg112")
	By("Creating custom storage class with non existing vg parameter", createVgPatternStorageClass)
	By("Creating and verifying PVC Bound status")
	createAndVerifyPVC(true)
	By("Verifying LVMVolume object to be Ready")
	VerifyLVMVolume(true, "lvmvg112")
	deleteAndVerifyPVC(pvcName)
	By("Verifying that PV doesnt exists after PVC deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func vgPatternNoMatchPresentTest() {
	device := setupVg(20, "lvmvg212")
	device_1 := setupVg(20, "lvmvg")
	defer cleanupVg(device_1, "lvmvg")
	defer cleanupVg(device, "lvmvg212")
	By("Creating custom storage class with non existing vg parameter", createVgPatternStorageClass)
	By("Creating and verifying PVC Not Bound status")
	createAndVerifyPVC(false)
	By("Verifying LVMVolume object to be Not Ready")
	VerifyLVMVolume(false, "")
	deleteAndVerifyPVC(pvcName)
	By("Verifying that PV doesnt exists after PVC deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func vgSpecifiedNotPresentTest() {
	device := setupVg(40, "lvmvg")
	defer cleanupVg(device, "lvmvg")
	By("Creating custom storage class with non existing vg parameter", createStorageClassWithNonExistingVg)
	By("creating and verifying PVC Not Bound status")
	createAndVerifyPVC(false)
	By("Verifying LVMVolume object to be Not Ready")
	VerifyLVMVolume(false, "")
	By("Deleting pvc")
	deleteAndVerifyPVC(pvcName)
	By("Verifying that PV doesnt exists after PVC deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func sharedVolumeTest() {
	By("Creating shared LV storage class", createSharedVolStorageClass)
	By("creating and verifying PVC bound status")
	createAndVerifyPVC(true)
	//we use two fio app pods for this test.
	appNames = append(appNames, "fio-ci-1")
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("Verifying LVMVolume object to be Not Ready")
	VerifyLVMVolume(true, "")
	By("Online resizing the shared volume")
	resizeAndVerifyPVC(true, "8Gi")
	deleteAppAndPvc(appNames, pvcName)
	By("Verifying that PV doesnt exists after PVC deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
	// Reset the app list back to original
	appNames = appNames[:len(appNames)-1]
}

func thinVolCreationTest() {
	By("Creating thinProvision storage class", createThinStorageClass)
	By("creating and verifying PVC bound status")
	createAndVerifyPVC(true)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object")
	VerifyLVMVolume(true, "")
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
	By("creating and verifying PVC bound status")
	createAndVerifyPVC(true)
	By("enabling monitoring on thinpool", enableThinpoolMonitoring)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying thinpool auto-extended", VerifyThinpoolExtend)
	By("verifying LVMVolume object")
	VerifyLVMVolume(true, "")
	deleteAppAndPvc(appNames, pvcName)
	By("Verifying that PV doesnt exists after PVC deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting thinProvision storage class", deleteStorageClass)
}

func sizedSnapFSTest() {
	createFstypeStorageClass("ext4")
	By("creating and verifying PVC bound status")
	createAndVerifyPVC(true)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object")
	VerifyLVMVolume(true, "")
	createSnapshot(pvcName, snapName, sizedsnapYAML)
	verifySnapshotCreated(snapName)
	deleteAppAndPvc(appNames, pvcName)
	By("Verifying that PV exists before Snapshot deletion")
	verifyPVForPVC(true, pvcName)
	deleteSnapshot(pvcName, snapName, sizedsnapYAML)
	By("Verifying that PV doesnt exists after Snapshot deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func sizedSnapBlockTest() {
	By("Creating default storage class", createStorageClass)
	By("creating and verifying PVC bound status")
	createAndVerifyPVC(true)
	By("Creating and deploying app pod", createDeployVerifyApp)
	By("verifying LVMVolume object")
	VerifyLVMVolume(true, "")
	createSnapshot(pvcName, snapName, sizedsnapYAML)
	verifySnapshotCreated(snapName)
	deleteAppAndPvc(appNames, pvcName)
	By("Verifying that PV exists before Snapshot deletion")
	verifyPVForPVC(true, pvcName)
	deleteSnapshot(pvcName, snapName, sizedsnapYAML)
	By("Verifying that PV doesnt exists after Snapshot deletion")
	verifyPVForPVC(false, pvcName)
	By("Deleting storage class", deleteStorageClass)
}

func sizedSnapshotTest() {
	By("Sized snapshot for filesystem volume", sizedSnapFSTest)
	By("Sized snapshot for block volume", sizedSnapBlockTest)
}

func leakProtectionTest() {
	By("Creating default storage class", createStorageClass)
	ds := deleteNodeDaemonSet() // ensure that provisioning remains in pending state.

	time.Sleep(30 * time.Second)

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

func volumeCreationTest() {
	device := setupVg(40, "lvmvg")
	defer cleanupVg(device, "lvmvg")
	By("###Running filesystem volume creation test###", fsVolCreationTest)
	By("###Running block volume creation test###", blockVolCreationTest)
	By("###Running thin volume creation test###", thinVolCreationTest)
	By("###Running leak protection test###", leakProtectionTest)
	By("###Running shared volume for two app pods on same node test###", sharedVolumeTest)
}

func schedulingTest() {
	By("###Running vg extend needed to provision test###", vgExtendNeededForProvsioningTest)
	By("###Running vg specified in sc not present test###", vgSpecifiedNotPresentTest)
	By("###Running lvmnode has vg matching vgpattern test###", vgPatternMatchPresentTest)
	By("###Running lvmnode doesnt have vg matching vgpattern test###", vgPatternNoMatchPresentTest)
}

func capacityTest() {
	device := setupVg(40, "lvmvg")
	defer cleanupVg(device, "lvmvg")
	By("###Running thin volume capacity test###", thinVolCapacityTest)
	By("###Running sized snapshot test###", sizedSnapshotTest)
}
