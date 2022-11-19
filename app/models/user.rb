# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  include Profilable

  belongs_to :account
end
