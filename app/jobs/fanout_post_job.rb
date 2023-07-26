# typed: strict
# frozen_string_literal: true

require "google/cloud/pubsub"

class FanoutPostJob
  extend T::Sig

  include SuckerPunch::Job

  sig { params(post_id: String).void }
  def perform(post_id:)
    topic.publish_async({post_id:}.to_json)
  end

  private

  sig { returns(Google::Cloud::PubSub::Topic) }
  def topic
    Mewst::CloudPubsub.client.topic(Rails.configuration.mewst["pubsub_topic_name_fanout_post"])
  end
end
