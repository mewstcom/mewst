# typed: strict
# frozen_string_literal: true

class Latest::UserResource < Latest::ApplicationResource
  delegate :id, :locale, to: :user

  sig { params(user: User).void }
  def initialize(viewer:, profile:)
    @user = user
  end

  sig { returns(User) }
  attr_reader :user
  private :user
end
