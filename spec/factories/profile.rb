# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile do
    sequence(:atname) { |n| "test_#{n}" }
    profileable_type { ProfileableType::User.serialize }
    joined_at { Time.current }
  end
end
