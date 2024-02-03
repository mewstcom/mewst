# typed: strict
# frozen_string_literal: true

class CreateAccountUseCase < ApplicationUseCase
  class Result < T::Struct
    const :actor, Actor
  end

  sig { params(atname: String, email: String, locale: String, password: String, time_zone: String).returns(Result) }
  def call(atname:, email:, locale:, password:, time_zone:)
    current_time = Time.current
    profile = Profile.new(profileable_type: ProfileableType::User.serialize, atname:, joined_at: current_time)

    actor = ActiveRecord::Base.transaction do
      profile.save!
      user = profile.create_user!(email:, password:, locale:, time_zone:, signed_up_at: current_time)
      profile.actors.create!(user:)
    end

    Result.new(actor:)
  end
end
