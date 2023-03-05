# typed: strict
# frozen_string_literal: true

require "google/cloud/pubsub"

class Mewst::CloudPubsub
  extend T::Sig

  def self.client
    @client ||= new(
      project_id: ENV.fetch("MEWST_GOOGLE_CLOUD_PUBSUB_PROJECT_ID"),
      credentials: Google::Cloud::PubSub::Credentials.new(
        JSON.parse(ENV.fetch("MEWST_GOOGLE_CLOUD_PUBSUB_CREDENTIALS"))
      ),
    ).client
  end

  sig { params(project_id: String, credentials: Google::Cloud::PubSub::Credentials).void }
  def initialize(project_id:, credentials:)
    @project_id = project_id
    @credentials = credentials
  end

  def client
    @client ||= Google::Cloud::PubSub.new(project_id:, credentials:)
  end

  private

  attr_reader :project_id, :credentials
end
