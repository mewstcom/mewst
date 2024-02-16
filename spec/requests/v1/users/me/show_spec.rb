# typed: false
# frozen_string_literal: true

RSpec.xdescribe "GET /v1/users/me", type: :request, api_version: :v1 do
  context "正常系" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:user) { viewer.user }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "ユーザ情報が返ること" do
      get("/v1/users/me", headers:)

      expect(response).to have_http_status(:ok)
      assert_response_schema_confirm(200)

      expected = {
        user: build_user_resource(user:)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)
    end
  end
end
