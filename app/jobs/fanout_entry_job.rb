# typed: strict
# frozen_string_literal: true

require "google/cloud/pubsub"

class FanoutEntryJob
  extend T::Sig

  include SuckerPunch::Job

  sig { params(entry_id: String).void }
  def perform(entry_id:)
    topic.publish_async({entry_id:}.to_json)
  end

  private

  sig { returns(Google::Cloud::PubSub::Topic) }
  def topic
    Mewst::CloudPubsub.client.topic(ENV.fetch("MEWST_PUBSUB_TOPIC_NAME_FANOUT_ENTRY"))
  end
end
