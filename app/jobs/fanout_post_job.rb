# typed: strict
# frozen_string_literal: true

class FanoutPostJob
  extend T::Sig

  include SuckerPunch::Job

  sig { params(post_id: String).void }
  def perform(post_id:)
    topic.publish_async({post_id:}.to_json)
  end

  private

  def topic
    @topic ||= Mewst::CloudPubsub.client.topic(ENV.fetch("MEWST_PUBSUB_TOPIC_NAME_FANOUT_POST"))
  end
end
