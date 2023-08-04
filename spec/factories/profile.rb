# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile do
    sequence(:atname) { |n| "test_#{n}" }
    joined_at { Time.current }

    trait :for_user do
      association :profileable, factory: :user
    end

    trait :with_access_token_for_web do
      after(:create) do |profile|
        application = create(:oauth_application, :mewst_web)
        create(:oauth_access_token, resource_owner: profile, application:, user: profile.profileable)
      end
    end
  end
end
