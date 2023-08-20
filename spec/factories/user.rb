# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    profile { association :profile, actor_type: :user }
    sequence(:email) { |n| "test_#{n}@example.com" }
    password { "xxx" }
    locale { :ja }
    signed_up_at { Time.current }

    trait :with_access_token_for_web do
      after(:create) do |user|
        application = create(:oauth_application, :mewst_web)
        create(:oauth_access_token, resource_owner: user.profile, application:, user:)
      end
    end
  end
end
