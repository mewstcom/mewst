# typed: strict
# frozen_string_literal: true

module Authorizable
  extend T::Sig
  extend ActiveSupport::Concern

  include Pundit::Authorization

  sig { returns(Profile) }
  def pundit_user
    current_profile!
  end
end
