# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :actor, class: "ActorRecord" do
    user_record { association :user_record }
    profile_record { user_record.profile_record }

    trait :with_access_token_for_web do
      after(:create) do |actor|
        application = create(:oauth_application, :mewst_web)
        create(:oauth_access_token, application:, resource_owner: actor)
      end
    end
  end
end
