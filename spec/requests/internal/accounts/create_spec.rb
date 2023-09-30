# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/accounts", type: :request, api_version: :internal do
  context "`atname` が不正なとき" do
    it "`422` を返すこと" do
      post("/internal/accounts", params: {
        atname: "a" * 31,
        email: "test@example.com",
        locale: "ja",
        password: "password"
      })

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "Atname is too long (maximum is 30 characters)"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:unprocessable_entity)
      assert_response_schema_confirm(422)
    end
  end

  context "入力データが正しいとき" do
    before do
      create(:oauth_application, :mewst_web)
    end

    it "`201` を返すこと" do
      expect(User.count).to eq(0)
      expect(Profile.count).to eq(0)
      expect(Actor.count).to eq(0)
      expect(OauthAccessToken.count).to eq(0)

      post("/internal/accounts", params: {
        atname: "shimbaco",
        email: "shimbaco@example.com",
        locale: "ja",
        password: "password"
      })

      expect(User.count).to eq(1)
      expect(Profile.count).to eq(1)
      expect(Actor.count).to eq(1)
      expect(OauthAccessToken.count).to eq(1)
      actor = Actor.first
      user = actor.user
      profile = actor.profile
      oauth_access_token = actor.oauth_access_tokens.first

      expected = {
        account: {
          oauth_access_token: {
            token: oauth_access_token.token
          },
          profile: build_profile_resource(profile:, viewer_has_followed: false),
          user: {
            id: user.id,
            locale: user.locale
          }
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:created)
      assert_response_schema_confirm(201)
    end
  end
end
