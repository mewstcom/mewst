# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /latest/me", type: :request, api_version: :latest do
  context "when invalid atname" do
    let!(:user) { create(:user, :with_access_token_for_web) }
    let!(:profile) { user.profile }
    let!(:oauth_access_token) { profile.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "responses 422" do
      patch("/latest/me", headers:, params: {
        atname: "invalid.atname",
        avatar_url: "https://example.com/avatar.png",
        description: "Hello",
        name: "John"
      })
      expect(response).to have_http_status(:unprocessable_entity)

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "Atname is invalid"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(422)
    end
  end

  context "when valid input data" do
    let!(:user) { create(:user, :with_access_token_for_web) }
    let!(:profile) { user.profile }
    let!(:oauth_access_token) { profile.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      profile.update!(atname: "before_update")
    end

    it "responses 200" do
      expect(profile.atname).to eq("before_update")
      expect(profile.avatar_url).to eq("")
      expect(profile.description).to eq("")
      expect(profile.name).to eq("")

      patch("/latest/me", headers:, params: {
        atname: "after_atname",
        avatar_url: "https://example.com/avatar.png",
        description: "Hello",
        name: "John"
      })
      expect(response).to have_http_status(:ok)

      expect(Profile.count).to eq(1)
      profile = Profile.first

      expect(profile.atname).to eq("after_atname")
      expect(profile.avatar_url).to eq("https://example.com/avatar.png")
      expect(profile.description).to eq("Hello")
      expect(profile.name).to eq("John")

      expected = {
        profile: {
          atname: profile.atname,
          avatar_url: profile.avatar_url,
          name: profile.name,
          description: profile.description,
          viewer_has_followed: false
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(200)
    end
  end
end
