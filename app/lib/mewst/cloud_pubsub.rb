# typed: strict
# frozen_string_literal: true

require "google/cloud/pubsub"

class Mewst::CloudPubsub
  extend T::Sig

  sig { returns(Google::Cloud::PubSub::Project) }
  def self.client
    new(
      project_id: ENV.fetch("MEWST_GOOGLE_CLOUD_PROJECT_ID"),
      credentials: Google::Cloud::PubSub::Credentials.new(
        JSON.parse(ENV.fetch("MEWST_GOOGLE_CLOUD_CREDENTIALS"))
      )
    ).client
  end

  sig { params(project_id: String, credentials: Google::Cloud::PubSub::Credentials).void }
  def initialize(project_id:, credentials:)
    @project_id = project_id
    @credentials = credentials
  end

  sig { returns(Google::Cloud::PubSub::Project) }
  def client
    Google::Cloud::PubSub.new(project_id:, credentials:)
  end

  private

  sig { returns(String) }
  attr_reader :project_id

  sig { returns(Google::Cloud::PubSub::Credentials) }
  attr_reader :credentials
end
