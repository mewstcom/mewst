# typed: strict
# frozen_string_literal: true

class Latest::UserResource < Latest::ApplicationResource
  delegate :id, :locale, :time_zone, to: :user

  sig { params(user: User).void }
  def initialize(user:)
    @user = user
  end

  sig { returns(User) }
  attr_reader :user
  private :user
end
