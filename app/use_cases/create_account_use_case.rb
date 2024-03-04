# typed: strict
# frozen_string_literal: true

class CreateAccountUseCase < ApplicationUseCase
  class Result < T::Struct
    const :actor, Actor
  end

  sig { params(atname: String, email: String, locale: String, password: String, time_zone: String).returns(Result) }
  def call(atname:, email:, locale:, password:, time_zone:)
    current_time = Time.current

    actor = ActiveRecord::Base.transaction do
      profile = Profile.create!(owner_type: ProfileOwnerType::User.serialize, atname:, joined_at: current_time)
      user = User.create!(email:, password:, locale:, time_zone:, signed_up_at: current_time)
      UserProfile.create!(user:, profile:)

      Actor.create!(user:, profile:)
    end

    Result.new(actor:)
  end
end
