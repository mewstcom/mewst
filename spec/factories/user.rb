# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    profile { association :profile, profileable_type: :user }
    sequence(:email) { |n| "test_#{n}@example.com" }
    password { "xxx" }
    locale { :ja }
    time_zone { "Asia/Tokyo" }
    signed_up_at { Time.current }
  end
end
