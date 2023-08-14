# typed: strict
# frozen_string_literal: true

class Latest::ApplicationController < ActionController::API
  extend T::Sig

  before_action :doorkeeper_authorize!

  private

  sig { returns(T.nilable(Profile)) }
  def current_profile
    @current_profile ||= T.let(doorkeeper_token&.profile, T.nilable(Profile))
  end

  sig { returns(T.nilable(User)) }
  def current_user
    @current_user ||= T.let(doorkeeper_token&.user, T.nilable(User))
  end
end
