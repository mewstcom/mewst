# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    sequence(:email) { |n| "test_#{n}@example.com" }
    password { "xxx" }
    locale { :ja }
    signed_up_at { Time.current }

    trait :with_profile do
      after(:create) do |user|
        create(:profile_member, user:)
      end
    end

    trait :with_mewst_web_access_token do
      after(:create) do |user|
        application = create(:oauth_application, :mewst_web)
        create(:oauth_access_token, resource_owner: user, application:)
      end
    end
  end
end
