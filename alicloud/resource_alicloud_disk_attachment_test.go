package alicloud

import (
	"fmt"
	"testing"
	"time"

	"strings"

	"github.com/alibaba/terraform-provider/alicloud/connectivity"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAlicloudDiskAttachment(t *testing.T) {
	var i ecs.Instance
	var v ecs.Disk

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_disk_attachment.disk-att",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDiskAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDiskAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"alicloud_instance.instance", &i),
					testAccCheckDiskExists(
						"alicloud_disk.disk", &v),
					testAccCheckDiskAttachmentExists(
						"alicloud_disk_attachment.disk-att", &i, &v),
				),
			},
		},
	})

}

func TestAccAlicloudDiskMultiAttachment(t *testing.T) {
	var i ecs.Instance
	var v ecs.Disk

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_disk_attachment.disks-attach.0",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDiskAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMultiDiskAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"alicloud_instance.instance", &i),
					testAccCheckDiskExists(
						"alicloud_disk.disks.0", &v),
					testAccCheckDiskAttachmentExists(
						"alicloud_disk_attachment.disks-attach.0", &i, &v),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_disk_attachment.disks-attach.1",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDiskAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccMultiDiskAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"alicloud_instance.instance", &i),
					testAccCheckDiskExists(
						"alicloud_disk.disks.1", &v),
					testAccCheckDiskAttachmentExists(
						"alicloud_disk_attachment.disks-attach.1", &i, &v),
				),
			},
		},
	})

}

func testAccCheckDiskAttachmentExists(n string, instance *ecs.Instance, disk *ecs.Disk) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Disk ID is set")
		}

		client := testAccProvider.Meta().(*connectivity.AliyunClient)
		ecsService := EcsService{client}

		return resource.Retry(3*time.Minute, func() *resource.RetryError {
			d, err := ecsService.DescribeDiskById(instance.InstanceId, rs.Primary.Attributes["disk_id"])
			if err != nil {
				return resource.NonRetryableError(err)
			}
			if d.Status != string(DiskInUse) {
				return resource.RetryableError(fmt.Errorf("Disk is in attaching - trying again while it attaches"))
			}

			*disk = d
			return nil
		})
	}
}

func testAccCheckDiskAttachmentDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_disk_attachment" {
			continue
		}
		// Try to find the Disk
		client := testAccProvider.Meta().(*connectivity.AliyunClient)
		ecsService := EcsService{client}
		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)
		disk, err := ecsService.DescribeDiskById(split[1], split[0])

		if err != nil {
			if NotFoundError(err) {
				continue
			}
			return fmt.Errorf("Describing disk %s got an error.", rs.Primary.ID)
		}
		if disk.Status != string(Available) {
			return fmt.Errorf("Error ECS Disk Attachment still exist")
		}
	}

	return nil
}

const testAccDiskAttachmentConfig = `
data "alicloud_zones" "default" {
	 available_disk_category = "cloud_ssd"
}

data "alicloud_instance_types" "default" {
 	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	cpu_core_count = 1
	memory_size = 2
}

data "alicloud_images" "default" {
        name_regex = "^ubuntu_14.*_64"
	most_recent = true
	owners = "system"
}

variable "name" {
	default = "tf-testAccDiskAttachmentConfig"
}

resource "alicloud_vpc" "vpc" {
	name = "${var.name}",
	cidr_block = "192.168.0.0/16"
}

resource "alicloud_vswitch" "vswitch" {
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	cidr_block = "192.168.0.0/24"
	vpc_id = "${alicloud_vpc.vpc.id}"
	name = "${var.name}"
}

resource "alicloud_security_group" "group" {
	name = "${var.name}"
	description = "foo"
    	vpc_id = "${alicloud_vpc.vpc.id}"
}

resource "alicloud_disk" "disk" {
  availability_zone = "${data.alicloud_zones.default.zones.0.id}"
  size = "50"
  name = "${var.name}"

  tags {
    Name = "TerraformTest-disk"
  }
}

resource "alicloud_instance" "instance" {
	image_id = "${data.alicloud_images.default.images.0.id}"
 	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	system_disk_category = "cloud_ssd"
	system_disk_size = 40
	instance_type = "${data.alicloud_instance_types.default.instance_types.0.id}"
	security_groups = ["${alicloud_security_group.group.id}"]
	instance_name = "${var.name}"
	vswitch_id = "${alicloud_vswitch.vswitch.id}"
}

resource "alicloud_disk_attachment" "disk-att" {
  disk_id = "${alicloud_disk.disk.id}"
  instance_id = "${alicloud_instance.instance.id}"
}
`
const testAccMultiDiskAttachmentConfig = `
data "alicloud_zones" "default" {
	 available_disk_category = "cloud_ssd"
}

data "alicloud_instance_types" "default" {
 	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	cpu_core_count = 1
	memory_size = 2
}

data "alicloud_images" "default" {
        name_regex = "^ubuntu_14.*_64"
	most_recent = true
	owners = "system"
}

variable "name" {
	default = "tf-testAccDiskAttachmentConfig"
}

variable "count" {
	default = "2"
}

resource "alicloud_vpc" "vpc" {
	name = "${var.name}",
	cidr_block = "192.168.0.0/16"
}

resource "alicloud_vswitch" "vswitch" {
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	cidr_block = "192.168.0.0/24"
	vpc_id = "${alicloud_vpc.vpc.id}"
	name = "${var.name}"
}

resource "alicloud_disk" "disks" {
	name = "${var.name}-${count.index}"
	count = "${var.count}"
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	size = "50"

	tags {
		Name = "TerraformTest-disk-${count.index}"
	}
}

resource "alicloud_instance" "instance" {
	image_id = "${data.alicloud_images.default.images.0.id}"
 	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	system_disk_category = "cloud_ssd"
	system_disk_size = 40
	instance_type = "${data.alicloud_instance_types.default.instance_types.0.id}"
	security_groups = ["${alicloud_security_group.group.id}"]
	instance_name = "${var.name}"
	vswitch_id = "${alicloud_vswitch.vswitch.id}"
}

resource "alicloud_disk_attachment" "disks-attach" {
	count = "${var.count}"
	disk_id     = "${element(alicloud_disk.disks.*.id, count.index)}"
	instance_id = "${alicloud_instance.instance.id}"
}

resource "alicloud_security_group" "group" {
	name = "${var.name}"
	description = "New security group"
	vpc_id = "${alicloud_vpc.vpc.id}"
}
`
