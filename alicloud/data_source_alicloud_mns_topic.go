package alicloud

import (
	"fmt"

	"github.com/alibaba/terraform-provider/alicloud/connectivity"
	"github.com/dxh031/ali_mns"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAlicloudMNSTopics() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAlicloudMNSTopicRead,
		Schema: map[string]*schema.Schema{
			"name_prefix": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"output_file": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Computed values
			"topics": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"maximum_message_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"logging_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAlicloudMNSTopicRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)

	var namePrefix string
	if v, ok := d.GetOk("name_prefix"); ok {
		namePrefix = v.(string)
	}

	var topicAttr []ali_mns.TopicAttribute
	for {
		var nextMaker string
		raw, err := client.WithMnsTopicManager(func(topicManager ali_mns.AliTopicManager) (interface{}, error) {
			return topicManager.ListTopicDetail(nextMaker, 1000, namePrefix)
		})
		if err != nil {
			return fmt.Errorf("Get topicDetails error: %#v", err)
		}
		topicDetails, _ := raw.(ali_mns.TopicDetails)
		for _, attr := range topicDetails.Attrs {
			topicAttr = append(topicAttr, attr)
		}
		nextMaker = topicDetails.NextMarker
		if nextMaker == "" {
			break
		}
	}

	return mnsTopicDescription(d, topicAttr)
}

func mnsTopicDescription(d *schema.ResourceData, topicAttr []ali_mns.TopicAttribute) error {
	var ids []string
	var s []map[string]interface{}

	for _, item := range topicAttr {
		mapping := map[string]interface{}{
			"id":                   item.TopicName,
			"name":                 item.TopicName,
			"maximum_message_size": item.MaxMessageSize,
			"logging_enabled":      item.LoggingEnabled,
		}

		ids = append(ids, item.TopicName)
		s = append(s, mapping)
	}

	d.SetId(dataResourceIdHash(ids))

	if err := d.Set("topics", s); err != nil {
		return err
	}
	if output, ok := d.GetOk("output_file"); ok && output.(string) != "" {
		writeToFile(output.(string), s)
	}
	return nil

}