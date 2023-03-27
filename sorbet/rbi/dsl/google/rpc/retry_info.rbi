# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Google::Rpc::RetryInfo`.
# Please instead update this file by running `bin/tapioca dsl Google::Rpc::RetryInfo`.

class Google::Rpc::RetryInfo
  sig { params(retry_delay: T.nilable(Google::Protobuf::Duration)).void }
  def initialize(retry_delay: nil); end

  sig { void }
  def clear_retry_delay; end

  sig { returns(T.nilable(Google::Protobuf::Duration)) }
  def retry_delay; end

  sig { params(value: T.nilable(Google::Protobuf::Duration)).void }
  def retry_delay=(value); end
end