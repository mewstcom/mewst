# typed: strict
# frozen_string_literal: true

class Follow < ApplicationRecord
  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"
end
