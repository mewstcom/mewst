# typed: strict
# frozen_string_literal: true

class CreateSessionUseCase < ApplicationUseCase
  class Result < T::Struct
    const :session, SessionRecord
  end

  sig { params(actor: ActorRecord, ip_address: T.nilable(String), user_agent: T.nilable(String)).returns(Result) }
  def call(actor:, ip_address:, user_agent:)
    session = ActiveRecord::Base.transaction do
      actor.sessions.start!(ip_address:, user_agent:)
    end

    Result.new(session:)
  end
end
