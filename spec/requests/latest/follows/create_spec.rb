# typed: false
# frozen_string_literal: true

RSpec.describe "POST /latest/@:atname/follow", type: :request, api_version: :latest do
  context "when invalid atname" do
    let!(:viewer) { create(:user, :with_access_token_for_web).profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 422" do
      post("/latest/@#{viewer.atname}/follow", headers:)
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "atname is myself. Cannot follow myself"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "when valid input data" do
    let!(:viewer) { create(:user, :with_access_token_for_web).profile }
    let!(:target_profile) { create(:user).profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 201" do
      expect(Follow.count).to eq(0)

      post("/latest/@#{target_profile.atname}/follow", headers:)
      expect(response).to have_http_status(:created)

      expect(Follow.count).to eq(1)

      expected = {
        profile: {
          atname: target_profile.atname,
          avatar_url: target_profile.avatar_url,
          name: target_profile.name,
          viewer_has_followed: true
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(201)
    end
  end
end
