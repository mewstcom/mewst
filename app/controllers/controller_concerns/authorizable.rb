# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Authorizable
  extend T::Sig
  extend ActiveSupport::Concern

  include Pundit::Authorization

  sig(:final) { returns(ActorRecord) }
  def pundit_user
    current_viewer!
  end
end
