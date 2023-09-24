# typed: strict
# frozen_string_literal: true

class Latest::ApplicationController < ActionController::API
  extend T::Sig

  before_action :doorkeeper_authorize!
end
