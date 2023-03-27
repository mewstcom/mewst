# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Google::Cloud::Tasks::V2::Queue`.
# Please instead update this file by running `bin/tapioca dsl Google::Cloud::Tasks::V2::Queue`.

class Google::Cloud::Tasks::V2::Queue
  sig do
    params(
      app_engine_routing_override: T.nilable(Google::Cloud::Tasks::V2::AppEngineRouting),
      name: T.nilable(String),
      purge_time: T.nilable(Google::Protobuf::Timestamp),
      rate_limits: T.nilable(Google::Cloud::Tasks::V2::RateLimits),
      retry_config: T.nilable(Google::Cloud::Tasks::V2::RetryConfig),
      stackdriver_logging_config: T.nilable(Google::Cloud::Tasks::V2::StackdriverLoggingConfig),
      state: T.nilable(T.any(Symbol, Integer))
    ).void
  end
  def initialize(app_engine_routing_override: nil, name: nil, purge_time: nil, rate_limits: nil, retry_config: nil, stackdriver_logging_config: nil, state: nil); end

  sig { returns(T.nilable(Google::Cloud::Tasks::V2::AppEngineRouting)) }
  def app_engine_routing_override; end

  sig { params(value: T.nilable(Google::Cloud::Tasks::V2::AppEngineRouting)).void }
  def app_engine_routing_override=(value); end

  sig { void }
  def clear_app_engine_routing_override; end

  sig { void }
  def clear_name; end

  sig { void }
  def clear_purge_time; end

  sig { void }
  def clear_rate_limits; end

  sig { void }
  def clear_retry_config; end

  sig { void }
  def clear_stackdriver_logging_config; end

  sig { void }
  def clear_state; end

  sig { returns(String) }
  def name; end

  sig { params(value: String).void }
  def name=(value); end

  sig { returns(T.nilable(Google::Protobuf::Timestamp)) }
  def purge_time; end

  sig { params(value: T.nilable(Google::Protobuf::Timestamp)).void }
  def purge_time=(value); end

  sig { returns(T.nilable(Google::Cloud::Tasks::V2::RateLimits)) }
  def rate_limits; end

  sig { params(value: T.nilable(Google::Cloud::Tasks::V2::RateLimits)).void }
  def rate_limits=(value); end

  sig { returns(T.nilable(Google::Cloud::Tasks::V2::RetryConfig)) }
  def retry_config; end

  sig { params(value: T.nilable(Google::Cloud::Tasks::V2::RetryConfig)).void }
  def retry_config=(value); end

  sig { returns(T.nilable(Google::Cloud::Tasks::V2::StackdriverLoggingConfig)) }
  def stackdriver_logging_config; end

  sig { params(value: T.nilable(Google::Cloud::Tasks::V2::StackdriverLoggingConfig)).void }
  def stackdriver_logging_config=(value); end

  sig { returns(T.any(Symbol, Integer)) }
  def state; end

  sig { params(value: T.any(Symbol, Integer)).void }
  def state=(value); end
end