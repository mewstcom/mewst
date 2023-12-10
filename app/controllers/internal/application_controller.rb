# typed: strict
# frozen_string_literal: true

class Internal::ApplicationController < ActionController::API
  extend T::Sig

  sig { returns(T.nilable(Actor)) }
  def current_viewer
    nil
  end
end
