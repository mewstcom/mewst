# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile_member do
    profile
    user
    joined_at { Time.current }
  end
end
