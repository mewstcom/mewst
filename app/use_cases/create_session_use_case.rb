# typed: strict
# frozen_string_literal: true

class CreateSessionUseCase < ApplicationUseCase
  class Result < T::Struct
    const :actor, Actor
  end

  sig { params(user: User).returns(Result) }
  def call(user:)
    ActiveRecord::Base.transaction do
      user.track_sign_in
    end

    Result.new(actor: user.first_actor)
  end
end
