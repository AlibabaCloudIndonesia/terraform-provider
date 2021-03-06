package alicloud

import (
	"fmt"
	"log"
	"testing"

	"strings"
	"time"

	"github.com/alibaba/terraform-provider/alicloud/connectivity"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func init() {
	resource.AddTestSweepers("alicloud_oss_bucket", &resource.Sweeper{
		Name: "alicloud_oss_bucket",
		F:    testSweepOSSBuckets,
	})
}

func testSweepOSSBuckets(region string) error {
	rawClient, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting Alicloud client: %s", err)
	}
	client := rawClient.(*connectivity.AliyunClient)

	prefixes := []string{
		"tf-testacc",
		"tf-test-",
		"test-bucket-",
		"tf-oss-test-",
		"tf-object-test-",
		"test-acc-alicloud-",
	}

	raw, err := client.WithOssClient(func(ossClient *oss.Client) (interface{}, error) {
		return ossClient.ListBuckets()
	})
	if err != nil {
		return fmt.Errorf("Error retrieving OSS buckets: %s", err)
	}
	resp, _ := raw.(oss.ListBucketsResult)
	sweeped := false

	for _, v := range resp.Buckets {
		name := v.Name
		skip := true
		for _, prefix := range prefixes {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				skip = false
				break
			}
		}
		if skip {
			log.Printf("[INFO] Skipping OSS bucket: %s", name)
			continue
		}
		sweeped = true
		raw, err := client.WithOssClient(func(ossClient *oss.Client) (interface{}, error) {
			return ossClient.Bucket(name)
		})
		if err != nil {
			return fmt.Errorf("Error getting bucket (%s): %#v", name, err)
		}
		bucket, _ := raw.(*oss.Bucket)
		if objects, err := bucket.ListObjects(); err != nil {
			log.Printf("[ERROR] Failed to list objects: %s", err)
		} else if len(objects.Objects) > 0 {
			for _, o := range objects.Objects {
				if err := bucket.DeleteObject(o.Key); err != nil {
					log.Printf("[ERROR] Failed to delete object (%s): %s.", o.Key, err)
				}
			}

		}

		log.Printf("[INFO] Deleting OSS bucket: %s", name)

		_, err = client.WithOssClient(func(ossClient *oss.Client) (interface{}, error) {
			return nil, ossClient.DeleteBucket(name)
		})
		if err != nil {
			log.Printf("[ERROR] Failed to delete OSS bucket (%s): %s", name, err)
		}
	}
	if sweeped {
		time.Sleep(5 * time.Second)
	}
	return nil
}

func TestAccAlicloudOssBucketBasic(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.basic",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAlicloudOssBucketBasicConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.basic", &bucket),
					resource.TestCheckResourceAttrSet(
						"alicloud_oss_bucket.basic",
						"location"),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.basic",
						"acl",
						"public-read"),
				),
			},
		},
	})

}

func TestAccAlicloudOssBucketCors(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.cors",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAlicloudOssBucketCorsConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.cors", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.cors",
						"cors_rule.#",
						"2"),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.cors",
						"cors_rule.0.allowed_headers.0",
						"authorization"),
				),
			},
		},
	})
}

func TestAccAlicloudOssBucketWebsite(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.website",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAlicloudOssBucketWebsiteConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.website", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.website",
						"website.#",
						"1"),
				),
			},
		},
	})
}
func TestAccAlicloudOssBucketLogging(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.logging",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAlicloudOssBucketLoggingConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.target", &bucket),
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.logging", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.logging",
						"logging.#",
						"1"),
				),
			},
		},
	})
}

func TestAccAlicloudOssBucketReferer(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.referer",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAlicloudOssBucketRefererConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.referer", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.referer",
						"referer_config.#",
						"1"),
				),
			},
		},
	})
}
func TestAccAlicloudOssBucketLifecycle(t *testing.T) {
	var bucket oss.BucketInfo

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_oss_bucket.lifecycle",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOssBucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAlicloudOssBucketLifecycleConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOssBucketExists(
						"alicloud_oss_bucket.lifecycle", &bucket),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.lifecycle",
						"lifecycle_rule.#",
						"2"),
					resource.TestCheckResourceAttr(
						"alicloud_oss_bucket.lifecycle",
						"lifecycle_rule.0.id",
						"rule1"),
				),
			},
		},
	})
}
func testAccCheckOssBucketExists(n string, b *oss.BucketInfo) resource.TestCheckFunc {
	providers := []*schema.Provider{testAccProvider}
	return testAccCheckOssBucketExistsWithProviders(n, b, &providers)
}
func testAccCheckOssBucketExistsWithProviders(n string, b *oss.BucketInfo, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		for _, provider := range *providers {
			// Ignore if Meta is empty, this can happen for validation providers
			if provider.Meta() == nil {
				continue
			}

			client := provider.Meta().(*connectivity.AliyunClient)
			ossService := OssService{client}
			bucket, err := ossService.QueryOssBucketById(rs.Primary.ID)
			log.Printf("[WARN]get oss bucket %#v", bucket)
			if err == nil && bucket != nil {
				*b = *bucket
				return nil
			}

			// Verify the error is what we want
			e, _ := err.(*oss.ServiceError)
			if e.Code == OssBucketNotFound {
				continue
			}
			if err != nil {
				return err

			}
		}

		return fmt.Errorf("Bucket not found")
	}
}

func TestResourceAlicloudOssBucketAcl_validation(t *testing.T) {
	_, errors := validateOssBucketAcl("incorrect", "acl")
	if len(errors) == 0 {
		t.Fatalf("Expected to trigger a validation error")
	}

	var testCases = []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "public-read",
			ErrCount: 0,
		},
		{
			Value:    "public-read-write",
			ErrCount: 0,
		},
	}

	for _, tc := range testCases {
		_, errors := validateOssBucketAcl(tc.Value, "acl")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected not to trigger a validation error")
		}
	}
}

func testAccCheckOssBucketDestroy(s *terraform.State) error {
	return testAccCheckOssBucketDestroyWithProvider(s, testAccProvider)
}

func testAccCheckOssBucketDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	client := provider.Meta().(*connectivity.AliyunClient)
	ossService := OssService{client}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_oss_bucket" {
			continue
		}

		// Try to find the resource
		bucket, err := ossService.QueryOssBucketById(rs.Primary.ID)
		if err == nil {
			if bucket.Name != "" {
				return fmt.Errorf("Found instance: %s", bucket.Name)
			}
		}

		// Verify the error is what we want
		if IsExceptedErrors(err, []string{OssBucketNotFound}) {
			continue
		}

		return err
	}

	return nil
}

func testAccAlicloudOssBucketBasicConfig(randInt int) string {
	return fmt.Sprintf(`
resource "alicloud_oss_bucket" "basic" {
	bucket = "tf-testacc-bucket-basic-%d"
	acl = "public-read"
}
`, randInt)
}

func testAccAlicloudOssBucketCorsConfig(randInt int) string {
	return fmt.Sprintf(`
resource "alicloud_oss_bucket" "cors" {
	bucket = "tf-testacc-bucket-cors-%d"
	cors_rule ={
		allowed_origins=["*"]
		allowed_methods=["PUT","GET"]
		allowed_headers=["authorization"]
	}
	cors_rule ={
		allowed_origins=["http://www.a.com", "http://www.b.com"]
		allowed_methods=["GET"]
		allowed_headers=["authorization"]
		expose_headers=["x-oss-test","x-oss-test1"]
		max_age_seconds=100
	}
}
`, randInt)
}

func testAccAlicloudOssBucketWebsiteConfig(randInt int) string {
	return fmt.Sprintf(`
resource "alicloud_oss_bucket" "website"{
	bucket = "tf-testacc-bucket-website-%d"
	website = {
		index_document = "index.html"
		error_document = "error.html"
	}
}
`, randInt)
}

func testAccAlicloudOssBucketLoggingConfig(randInt int) string {
	return fmt.Sprintf(`
resource "alicloud_oss_bucket" "target"{
	bucket = "tf-testacc-target-%d"
}
resource "alicloud_oss_bucket" "logging" {
	bucket = "tf-testacc-bucket-logging-%d"
	logging {
		target_bucket = "${alicloud_oss_bucket.target.id}"
		target_prefix = "log/"
	}
}
`, randInt, randInt)
}

func testAccAlicloudOssBucketRefererConfig(randInt int) string {
	return fmt.Sprintf(`
resource "alicloud_oss_bucket" "referer" {
	bucket = "tf-testacc-bucket-referer-%d"
	referer_config {
		allow_empty = false
		referers = ["http://www.aliyun.com", "https://www.aliyun.com"]
	}
}
`, randInt)
}

func testAccAlicloudOssBucketLifecycleConfig(randInt int) string {
	return fmt.Sprintf(`
resource "alicloud_oss_bucket" "lifecycle"{
	bucket = "tf-testacc-bucket-lifecycle-%d"
	lifecycle_rule {
		id = "rule1"
		prefix = "path1/"
		enabled = true
		expiration {
			days = 365
		}
	}
	lifecycle_rule {
		id = "rule2"
		prefix = "path2/"
		enabled = true
		expiration {
			date = "2018-01-12"
		}
	}
}
`, randInt)
}
