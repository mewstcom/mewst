# typed: strict
# frozen_string_literal: true

module Profilable
  extend ActiveSupport::Concern

  included do
    has_one :profile, as: :profilable
  end
end
