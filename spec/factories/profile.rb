# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile do
    sequence(:atname) { |n| "test_#{n}" }
    owner_type { ProfileOwnerType::User.serialize }
    joined_at { Time.current }
  end
end
