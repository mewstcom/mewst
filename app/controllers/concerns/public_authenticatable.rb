# typed: strict
# frozen_string_literal: true

module PublicAuthenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    before_action :doorkeeper_authorize!
  end

  sig { returns(T.nilable(Actor)) }
  def current_viewer
    @current_viewer ||= T.let(doorkeeper_token&.resource_owner, T.nilable(Actor))
  end

  sig { returns(Actor) }
  def current_viewer!
    current_viewer.not_nil!
  end
end
