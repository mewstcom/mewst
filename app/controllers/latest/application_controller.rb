# typed: strict
# frozen_string_literal: true

class Latest::ApplicationController < ActionController::API
  extend T::Sig

  before_action :doorkeeper_authorize!

  private

  def current_user
    @current_user ||= doorkeeper_token&.user
  end
end
