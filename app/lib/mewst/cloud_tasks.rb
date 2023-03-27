# typed: strict
# frozen_string_literal: true

require "google/cloud/tasks"
require "google/cloud/tasks/v2"

class Mewst::CloudTasks
  extend T::Sig

  sig { params(priority: Symbol).void }
  def initialize(priority: :default)
    @priority = T.let(priority, Symbol)
  end

  sig { params(path: String, payload: T.nilable(String), seconds: T.nilable(Integer)).void }
  def create_task(path:, payload: nil, seconds: nil)
    parent = client.queue_path(
      project: ENV.fetch("MEWST_GOOGLE_CLOUD_PROJECT_ID"),
      location: ENV.fetch("MEWST_GOOGLE_CLOUD_TASKS_LOCATION"),
      queue: queue_id
    )
    url = "#{ENV.fetch("MEWST_URL")}#{path}"

    task = {
      http_request: {
        http_method: :POST,
        url:,
        oidc_token: {
          service_account_email: ENV.fetch("MEWST_GOOGLE_CLOUD_SERVICE_EMAIL")
        },
        headers: {
          "Content-Type": "application/json"
        }
      }
    }

    if payload
      task[:http_request][:body] = payload
    end

    if seconds
      timestamp = Google::Protobuf::Timestamp.new
      timestamp.seconds = Time.now.to_i + seconds.to_i
      task[:schedule_time] = timestamp
    end

    response = client.create_task(parent:, task:)

    Rails.logger.debug "Created task: #{response.name}"
  end

  private

  sig { returns(Symbol) }
  attr_reader :priority

  sig { returns(Google::Cloud::Tasks::V2::CloudTasks::Client) }
  def client
    Google::Cloud::Tasks.configure do |config|
      config.credentials = Google::Cloud::Tasks::V2::CloudTasks::Credentials.new(
        JSON.parse(ENV.fetch("MEWST_GOOGLE_CLOUD_CREDENTIALS"))
      )
    end

    Google::Cloud::Tasks.cloud_tasks
  end

  sig { returns(String) }
  def queue_id
    ENV.fetch("MEWST_GOOGLE_CLOUD_TASKS_QUEUE_ID_DEFAULT")
  end
end
