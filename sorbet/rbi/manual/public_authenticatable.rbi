# typed: strong
# frozen_string_literal: true

module ControllerConcerns::PublicAuthenticatable
  def self.before_action(*args)
  end

  def doorkeeper_token
  end
end
