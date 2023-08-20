# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile do
    sequence(:atname) { |n| "test_#{n}" }
    joined_at { Time.current }
  end
end
