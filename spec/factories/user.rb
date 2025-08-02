# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user_record, class: "UserRecord" do
    profile_record { association :profile_record, owner_type: ProfileOwnerType::User.serialize }
    sequence(:email) { |n| "test_#{n}@example.com" }
    password { "passw0rd" }
    locale { :ja }
    time_zone { "Asia/Tokyo" }
    signed_up_at { Time.current }
  end
end
