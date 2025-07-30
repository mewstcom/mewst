# typed: false
# frozen_string_literal: true

RSpec.xdescribe "POST /v1/internal/accounts", type: :request do
  context "`atname` が不正なとき" do
    let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
    let!(:headers) { {"HTTP_AUTHORIZATION" => token} }

    it "`422` を返すこと" do
      post("/v1/internal/accounts", headers:, params: {
        atname: "a" * (Profile::ATNAME_MAX_LENGTH + 1),
        email: "test@example.com",
        locale: "ja",
        password: "password"
      })

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "atname",
            message: "Atname is too long (maximum is 20 characters)"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:unprocessable_entity)
    end
  end

  context "入力データが正しいとき" do
    let!(:token) { ActionController::HttpAuthentication::Token.encode_credentials(Rails.configuration.mewst["internal_api_token"]) }
    let!(:headers) { {"HTTP_AUTHORIZATION" => token} }

    before do
      create(:oauth_application, :mewst_web)
    end

    it "`201` を返すこと" do
      expect(User.count).to eq(0)
      expect(Profile.count).to eq(0)
      expect(Actor.count).to eq(0)
      expect(OauthAccessToken.count).to eq(0)

      post("/v1/internal/accounts", headers:, params: {
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
      user = actor.user_record
      profile = actor.profile_record
      oauth_access_token = actor.oauth_access_tokens.first

      expected = {
        account: {
          oauth_access_token: {
            token: oauth_access_token.token
          },
          profile: build_profile_resource(profile:, viewer_has_followed: false),
          user: build_user_resource(user:)
        }
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:created)
    end
  end
end
