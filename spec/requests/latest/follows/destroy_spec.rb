# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /latest/@:atname/follow", type: :request, api_version: :latest do
  context "when invalid atname" do
    let!(:viewer) { create(:user, :with_access_token_for_web).profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 422" do
      delete("/latest/@unknown/follow", headers:)
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "atname not found"
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
    let!(:form) { Latest::FollowForm.new(viewer:, target_atname: target_profile.atname) }
    let!(:input) { FollowProfileService::Input.from_latest_form(form:) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      FollowProfileService.new.call(input:)
    end

    it "responses 200" do
      expect(Follow.count).to eq(1)

      delete("/latest/@#{target_profile.atname}/follow", headers:)
      expect(response).to have_http_status(:ok)

      expect(Follow.count).to eq(0)

      expected = {
        profile: {
          atname: target_profile.atname,
          avatar_url: target_profile.avatar_url,
          description: target_profile.description,
          name: target_profile.name,
          viewer_has_followed: false
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(200)
    end
  end
end
