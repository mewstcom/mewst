# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    sequence(:email) { |n| "test_#{n}@example.com" }
    password { "xxx" }
    locale { :ja }
    signed_up_at { Time.current }
  end
end
