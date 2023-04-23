# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile do
    sequence(:atname) { |n| "test_#{n}" }
    joined_at { Time.current }

    trait :with_account do
      after(:create) do |profile|
        create(:profile_member, profile:)
      end
    end
  end
end
