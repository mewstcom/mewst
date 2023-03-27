# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Google::Cloud::PubSub::V1::Encoding`.
# Please instead update this file by running `bin/tapioca dsl Google::Cloud::PubSub::V1::Encoding`.

module Google::Cloud::PubSub::V1::Encoding
  class << self
    sig { returns(Google::Protobuf::EnumDescriptor) }
    def descriptor; end

    sig { params(number: Integer).returns(T.nilable(Symbol)) }
    def lookup(number); end

    sig { params(symbol: Symbol).returns(T.nilable(Integer)) }
    def resolve(symbol); end
  end
end

Google::Cloud::PubSub::V1::Encoding::BINARY = 2
Google::Cloud::PubSub::V1::Encoding::ENCODING_UNSPECIFIED = 0
Google::Cloud::PubSub::V1::Encoding::JSON = 1