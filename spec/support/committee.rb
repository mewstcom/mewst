# typed: false
# frozen_string_literal: true

RSpec.configure do |config|
  config.include Committee::Rails::Test::Methods
  config.add_setting :committee_options
  config.committee_options = {
    query_hash_key: "rack.request.query_hash",
    parse_response_by_content_type: false,
    strict_reference_validation: true
  }

  config.before :all, api_version: :v1 do
    RSpec.configure do |config|
      config.committee_options[:schema_path] = Rails.root.join("openapi/v1.yaml").to_s
      config.committee_options[:prefix] = "/v1"
    end
  end
end
