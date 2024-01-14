# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /v1/profiles/me", type: :request, api_version: :v1 do
  context "アットネームが不正なとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    it "`422` を返すこと" do
      patch("/v1/profiles/me", headers:, params: {
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

  context "入力データが正しいとき" do
    let!(:viewer) { create(:actor, :with_access_token_for_web) }
    let!(:profile) { viewer.profile }
    let!(:oauth_access_token) { viewer.oauth_access_tokens.first }
    let!(:headers) { {"Authorization" => "bearer #{oauth_access_token.token}"} }

    before do
      profile.update!(atname: "before_update")
    end

    it "`200` を返すこと" do
      expect(profile.atname).to eq("before_update")
      expect(profile.avatar_url).to eq("")
      expect(profile.description).to eq("")
      expect(profile.name).to eq("")

      patch("/v1/profiles/me", headers:, params: {
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
        profile: build_profile_resource(profile:, viewer_has_followed: false)
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      assert_response_schema_confirm(200)
    end
  end
end
