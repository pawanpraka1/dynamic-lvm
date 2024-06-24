Feature: Validate vgpattern and volgroup based scheduling.
  Background:
      Given Single lvmnode cluster is configured
      And lvm vg "lvmvg" is created on a node

      Scenario: test sc with volgroup present on the node
          Given a sc is created with volgroup as "lvmvg"
          And pvc is created referencing the same sc
          Then lvmvolume cr should get created
          And lvmvolume cr state should be Ready

      Scenario: test sc with volgroup not present on the node
          Given a sc is created with volgroup as "lvmvgn"
          And pvc is created referencing the same sc
          Then pvc should be in Pending
          And lvmvolume cr should not get created

  
  Background:
      Given Single lvmnode cluster is configured
      And lvm vg "lvmvg" and "lvmvg1" is created on a node

      Scenario: test sc with vgpattern matching atleast one vg
          Given a sc is created with vgpattern "lvmvg1.*"
          When a pvc is created by referencing the sc
          Then pvc should be Bound
          And lvmvolume cr should be in "lvmvg1" vg

  
      Scenario: test sc with vgpattern matching no vg
          Given a sc is created with vgpattern "lvmvgn.*"
          When a pvc is created by referencing the sc
          Then pvc should not be Bound
          And lvmvolume cr should not be created

   Background:
        Given Single lvmnode cluster is configured
        And both <vg> are created with their respective <size>

        Scenario: test pvc with larger size then vg free space
            Given sc is created with volgroup parameter as "lvmvg"
            When a pvc is created with size "1.3G"
            Then pvc should not be bound
            And lvmvolume cr should not be created
            When "lvmvg" is extended to "1.5G"
            Then pvc should be Bound
            And lvmvolume cr should have expected vg value

            Examples:
                |  vg        | size     |
                | lvmvg      | 1G       |
                | lvmvg1     | 2G       |

  
  Background:
      Given a 3 lvmnode cluster is configured
      And each node <node> has <vg> created with same size 

      Scenario: test sc with volgroup present only on one node
          Given a sc is created with volgroup as "lvmvg1" with no topology
          When pvc is created referencing the same sc
          Then pvc should be in Pending state
          And lvmvolume cr should not be created

      Scenario: test sc with volgroup present on one node with allowed topology
          Given a sc is created with volgroup parameter as "lvmvg1"
          And sc has topology key with "node1" value
          When pvc is created referencing the same sc
          Then pvc should be bound 
          And lvmvolume cr should be Ready
          And lvmvolume cr should have expected owner and vg value

       Scenario: test sc with volgroup not present on any node
          Given a sc is created with volgroup parameter as "lvmvgn"
          When pvc is created referencing the same sc
          Then pvc should be in Pending state
          And lvmvolume cr should not be created

        Examples:
            |  node        | vg           |
            | node1        | lvmvg1       |
            | node2        | lvmvg2       |
            | node3        | lvmvg3       |
       

  Background:
      Given a 3 lvmnode cluster is configured
      And each node <node> has <vg> created with <size>

      Scenario: test sc with volgroup present on 2 nodes with varying free space and allowed topology
          Given a sc is created with volgroup parameter as "lvmvg"
          And node1 and node2 is in allowed topology
          When a pvc is created with "1.3G" size
          Then pvc should be Bound
          And lvmvolume cr should be in Ready state
          And lvmvolume cr should have expected owner and vg value

      Examples:
        |  node        | vg          |  size    |
        | node1        | lvmvg       |  1G      |
        | node2        | lvmvg       |  1.5G    |
        | node3        | lvmvgdiff   |  1G      |

        
      Scenario: test sc with volgroup present on 2 nodes with less space and allowed topology
          Given a sc is created with volgroup parameter as "lvmvg"
          And node1 and node2 is in allowed topology
          When a pvc is created with "1.3G" size
          Then pvc should be Pending
          And lvmvolume cr should not be created
          When node2 vg is extended to <1.5G>
          Then pvc should be Bound
          And lvmvolume cr should be created
          And lvmvolume cr should have expected owner and vg value

      Examples:
        |  node        | vg          |  size    |
        | node1        | lvmvg       |  1G      |
        | node2        | lvmvg       |  1G      |
        | node3        | lvmvgdiff   |  1G      |

  Background:

      Given a 3 lvmnode cluster is configured
      And each lvm node <node> has <vg> and <vg1> with <size> and <size1> respectively 

      Scenario: test sc with volgroup present on 2 nodes with varying free space and allowed topology
          Given a sc is created with volgroup parameter as “lvmvg“
          And using WaitForFirstConsumer as binding mode
          And pvc is created with size “1.5G“
          Then lvmvolume cr should not be created
          When an app is created referencing the pvc
          Then pvc should be created
          And App should be running
          And lvmvolume cr should be Ready
          And lvmvolume cr should have expected owner and vg value

       Examples:
            |  node      | vg             |  size      |  vg1       |   size1     |
            | node1      | lvmvg          |  1G        | lvmvg1     |     2G      |
            | node2      | lvmvg          |  1.6G      | lvmvg1     |     2G      |
            | node3      | lvmvgdiff      |  1G        | lvmvg1     |     2G      |
      
  Background:

      Given a 3 lvmnode cluster is configured
      And each node <node> has <vg> and <vg2> created with same size

      Scenario: test sc with vgpattern matching vg present in all node
          Given sc1 is created with vgpattern parameter as "lvmvg1.*"
          And sc2 is created with vgpattern parameter as "lvmvg2.*"
          And sc3 is created with vgpattern parameter as "lvmvg3*."
          When pvc1 is created referencing sc1
          And pvc2 is created referencing sc2
          And pvc3 is created referencing sc3
          Then pvc1 and pvc2 should be in Bound state
          And pvc1 and pvc2 lvmvolume cr should be in expected vg and node
          And pvc3 should be in Pending state

  
      Examples:
        |  node        | vg           | vg2           |
        | node1        | lvmvg11       | lvmvg21      |
        | node2        | lvmvg12       | lvmvg22      |
        | node3        | lvmvg13       | lvmvg23      |

  Background:
    Given a 3 node lvm localpv cluster is configured
    And only one node has vg named "lvmvg"

    Scenario: test sc with volgroup present on one node having a custom topology key label
        Given a sc is created with volgroup parameter as "lvmvg"
        And sc has a custom topology key "a-custom-key" having value "a-custom-value"
        When pvc is created referencing the same sc
        Then pvc should be pending
        When the lvm-node daemonset having vg is edited and made aware of the custom key
        Then pvc should be bound
        And lvmvolume cr should be Ready
        And lvmvolume cr should have expected owner and vg value
