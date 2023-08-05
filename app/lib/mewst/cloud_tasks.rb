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
      project: Rails.configuration.mewst["google_cloud_project_id"],
      location: Rails.configuration.mewst["google_cloud_tasks_location"],
      queue: queue_id
    )
    url = "#{Rails.configuration.mewst["url"]}#{path}"

    task = {
      http_request: {
        http_method: :POST,
        url:,
        oidc_token: {
          service_account_email: Rails.configuration.mewst["google_cloud_service_email"]
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

  sig { returns(Symbol) }
  attr_reader :priority
  private :priority

  sig { returns(Google::Cloud::Tasks::V2::CloudTasks::Client) }
  private def client
    Google::Cloud::Tasks.configure do |config|
      config.credentials = Google::Cloud::Tasks::V2::CloudTasks::Credentials.new(
        Rails.configuration.mewst["google_cloud_credentials"]
      )
    end

    Google::Cloud::Tasks.cloud_tasks
  end

  sig { returns(String) }
  private def queue_id
    Rails.configuration.mewst["google_cloud_tasks_queue_id_default"]
  end
end
