# typed: false
# frozen_string_literal: true

RSpec.describe "POST /internal/sessions", type: :request, api_version: :internal do
  context "when incorrect password" do
    let!(:email) { "test@example.com" }
    let!(:password) { "correct_password" }

    before do
      create(:user, :with_profile, email:, password:)
    end

    it "responses 422" do
      post("/internal/sessions", params: {
        email:,
        password: "incorrect_password"
      })

      expected = {
        errors: [
          {
            code: "invalid_input_data",
            field: "base",
            message: "Login failed. Email or password is incorrect"
          }
        ]
      }
      actual = JSON.parse(response.body)
      expect(actual).to include(expected.deep_stringify_keys)

      expect(response).to have_http_status(:unprocessable_entity)
      assert_response_schema_confirm(422)
    end
  end

  context "when correct email/password" do
    let!(:email) { "test@example.com" }
    let!(:password) { "correct_password" }
    let!(:user) { create(:user, :with_profile, :with_mewst_web_access_token, email:, password:) }
    let!(:profile) { user.profiles.first }
    let!(:oauth_access_token) { user.oauth_access_tokens.first }

    it "responses 201" do
      expect(user.sign_in_count).to eq(0)

      post("/internal/sessions", params: {
        email:,
        password:
      })
      user.reload

      expect(user.sign_in_count).to eq(1)

      expected = {
        account: {
          oauth_access_token: {
            token: oauth_access_token.token
          },
          profile: {
            atname: profile.atname,
            avatar_url: profile.avatar_url,
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
