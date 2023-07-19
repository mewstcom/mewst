# typed: false
# frozen_string_literal: true

RSpec.describe "GET /latest/timeline", type: :request, api_version: :latest do
  context "when success" do
    it "returns expected body schema" do
      get "/latest/@hoge/timeline"
      assert_response_schema_confirm(200)
    end

    it "returns expected status" do
      get "/latest/@hoge/timeline"
      expect(response).to have_http_status(:ok)
    end
  end
end
