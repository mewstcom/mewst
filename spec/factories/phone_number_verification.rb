# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :phone_number_verification do
    sequence(:phone_number) { |n| "+81900000000#{n}" }
    sequence(:phone_number_origin) { |n| "0900000000#{n}" }
    sequence(:confirmation_code) { |n| "00000#{n}" }
  end
end
