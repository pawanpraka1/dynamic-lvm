  Feature: Validate scheduling algorithm
    Background:
        Given 3 node lvmnode cluster is configured
        And lvm vg "lvmvg" with 20G is created on each node

            Scenario: test CapacityWeighted scheduler logic
                Given sc is created with scheduler parameter set to "CapacityWeighted"
                When "pvc-1" is created using size "5G"
                And "pvc-2" is created using size "5G"
                And "pvc-3" is created using size "2G"
                Then all three pvc lvmvolume should be placed on different nodes
                When "pvc-4" is created with size "1G"
                Then "pvc-4" lvmvolume should be placed on "pvc-3" lvmvolume node
                When "pvc-5" is created with size "1G"
                Then "pvc-5" lvmvolume should be placed on "pvc-4" lvmvolume node
                When "pvc-6" is created with size "2G"
                Then "pvc-6" lvmvolume should be placed on "pvc-5" lvmvolume node
                When "pvc-7" is created with size "3G"
                Then "pvc-7" lvmvolume should be placed on "pvc-1" or "pvc-2" lvmvolume node
                When "pvc-8" is created with size "3G"
                Then "pvc-8" lvmvolume should not be placed on "pvc-7" and "pvc-3" node

            Scenario: test SpaceWeighted with vgextend logic
                Given sc is created by not setting scheduler parameter explicitly
                When "pvc-1" is created with using size "6G"
                And "pvc-2" is created with using size "2G"
                And "pvc-3" is created with using size "3G"
                Then all pvc lvmvolume should be placed on different nodes
                When "pvc-1" vg is extended by 10G
                And "pvc-4" is created with size "5G" 
                Then "pvc-4" lvmvolume should be placed on "pvc-1" lvmvolume node

            Scenario: test SpaceWeighted without vgextend logic
                Given sc is created by not setting scheduler parameter explicitly
                When "pvc-1" is created with using size "6G"
                And "pvc-2" is created with using size "2G"
                And "pvc-3" is created with using size "3G"
                Then all pvc lvmvolume should be placed on different nodes
                When "pvc-4" is created with size "5G" 
                Then "pvc-4" lvmvolume should be placed on "pvc-2" lvmvolume node

            Scenario: test VolumeWighted logic
                Given sc is created with scheduler parameter set to "VolumeWeighted"
                When "pvc-1" is created with using size "6G"
                And "pvc-2" is created with using size "2G"
                And "pvc-3" is created with using size "3G"
                Then all pvc lvmvolume should be placed on different nodes
                When "pvc-4" is created with size "4G"
                Then "pvc-4" lvmvolume can be on any node
                When "pvc-5" is created with size "3G"
                Then "pvc-5" lvmvolume can not be on "pvc-4" node
                When "pvc-6" is created with size "6G"
                Then "pvc-6" can not be on "pvc-4" and "pvc-5" lvmvolume node
