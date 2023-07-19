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

  config.before :all, api_version: :internal do
    RSpec.configure do |config|
      config.committee_options[:schema_path] = Rails.root.join("app/controllers/internal/openapi.yaml").to_s
      config.committee_options[:prefix] = "/internal"
    end
  end

  config.before :all, api_version: :latest do
    RSpec.configure do |config|
      config.committee_options[:schema_path] = Rails.root.join("app/controllers/latest/openapi.yaml").to_s
      config.committee_options[:prefix] = "/latest"
    end
  end
end
