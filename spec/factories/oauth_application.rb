# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :oauth_application do
    sequence(:name) { |n| "OAuth App #{n}" }
    sequence(:uid) { |n| "oauth-app-#{n}" }
    secret { "secret" }
    redirect_uri { "https://example.com/callback" }
    scopes { "read" }

    trait :mewst_web do
      name { "Mewst for Web" }
      uid { OauthApplication::MEWST_WEB_UID }
    end
  end
end
