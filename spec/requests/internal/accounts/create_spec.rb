# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/accounts", type: :request, api_version: :internal do
  context "when invalid atname" do
    it "responses 422" do
      post("/internal/accounts", params: {
        atname: "a" * 31,
        email: "test@example.com",
        locale: "ja",
        password: "password"
      })

      expected = {
        errors: [
          {
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

  context "when valid input data" do
    before do
      create(:oauth_application, :mewst_web)
    end

    it "responses 201" do
      expect(User.count).to eq(0)
      expect(Profile.count).to eq(0)
      expect(OauthAccessToken.count).to eq(0)

      post("/internal/accounts", params: {
        atname: "shimbaco",
        email: "shimbaco@example.com",
        locale: "ja",
        password: "password"
      })

      expect(User.count).to eq(1)
      expect(Profile.count).to eq(1)
      expect(OauthAccessToken.count).to eq(1)
      user = User.first
      profile = user.profile
      oauth_access_token = profile.oauth_access_tokens.first

      expected = {
        account: {
          oauth_access_token: {
            token: oauth_access_token.token
          },
          profile: {
            atname: profile.atname,
            avatar_url: profile.avatar_url,
            description: profile.description,
            id: profile.id,
            name: profile.name
          },
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
