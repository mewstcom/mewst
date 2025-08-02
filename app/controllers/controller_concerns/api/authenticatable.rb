# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Api::Authenticatable
  # extend T::Sig
  # extend ActiveSupport::Concern
  #
  # included do
  #   before_action :doorkeeper_authorize!
  # end
  #
  # sig { returns(T.nilable(ActorRecord)) }
  # def current_viewer
  #   @current_viewer ||= T.let(doorkeeper_token&.resource_owner, T.nilable(ActorRecord))
  # end
  #
  # sig { returns(ActorRecord) }
  # def current_viewer!
  #   current_viewer.not_nil!
  # end
end
