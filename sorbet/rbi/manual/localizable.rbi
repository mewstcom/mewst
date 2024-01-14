# typed: strong
# frozen_string_literal: true

module ControllerConcerns::Localizable
  def current_viewer; end
  def http_accept_language; end
  def params; end
  def session; end
end
