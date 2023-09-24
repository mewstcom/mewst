# typed: strict
# frozen_string_literal: true

module Latest::Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_viewer
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
